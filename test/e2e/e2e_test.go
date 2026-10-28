// Package e2e builds both apps, runs them against a fresh database in a temp
// directory and walks the staff and customer flows over HTTP. Run it with
// `make test` (it needs the generated CSS, which the binaries embed).
package e2e

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	initKey = "1111111111111111"
	refNum  = "1234567890123456789012345"
)

// TestFlows shares one pair of servers: each step builds on the state the
// previous ones left, so a failure stops the run.
func TestFlows(t *testing.T) {
	dir := t.TempDir()
	build(t, dir, "STAFF", "staff")
	build(t, dir, "USER", "user")

	// Staff first: only it creates the DB.
	staffURL := start(t, dir, "staff")
	userURL := start(t, dir, "user")
	t.Cleanup(func() { dumpLogsOnFailure(t, dir) })

	admin := newClient(t, staffURL, "e2e-admin")
	cust := newClient(t, userURL, "e2e-customer")

	step(t, "init admin", func(t *testing.T) {
		admin.expect(t, 200, admin.get("/init", false))
		bad := initForm("2222222222222222")
		admin.expect(t, 401, admin.form("POST", "/init", bad, false))
		// Once staff exists, /init sends everyone away.
		res := admin.expect(t, 301, admin.form("POST", "/init", initForm(initKey), false))
		if loc := res.header.Get("Location"); loc != "/menu" { t.Errorf("Location = %q, want /menu", loc) }
		admin.expect(t, 301, admin.get("/init", false))
	})

	db := openDB(t, dir)
	seed(t, db)
	var available, soldOut int
	scan(t, db, `SELECT id FROM items WHERE name = 'Pan dulce'`, &available)
	scan(t, db, `SELECT id FROM items WHERE available = 0 LIMIT 1`, &soldOut)

	step(t, "admin login", func(t *testing.T) {
		admin.expect(t, 200, admin.form("POST", "/login", creds("admin", "admin1234"), false))
	})

	step(t, "menu hides sold-out items", func(t *testing.T) {
		res := cust.expect(t, 200, cust.get("/menu", false))
		res.contains(t, "Pan dulce")
		res.lacks(t, "Torta de chocolate")
	})

	step(t, "register", func(t *testing.T) {
		mismatch := url.Values{"username": {"cliente"}, "password-1": {"clave1234"}, "password-2": {"otra1234"}}
		cust.expect(t, 400, cust.form("POST", "/register", mismatch, false)).contains(t, "no coinciden")
		ok := url.Values{"username": {"cliente"}, "password-1": {"clave1234"}, "password-2": {"clave1234"}}
		cust.expect(t, 201, cust.form("POST", "/register", ok, true))
		ok.Set("username", "cliente2")
		cust.expect(t, 403, cust.form("POST", "/register", ok, true))
	})

	step(t, "customer login", func(t *testing.T) {
		cust.expect(t, 200, cust.form("POST", "/login", creds("cliente", "clave1234"), false))
		cust.expect(t, 200, cust.get("/cart", false)).contains(t, "Pan dulce")
	})

	step(t, "place order", func(t *testing.T) {
		cust.expect(t, 400, cust.order(map[int]int{available: 0}))
		cust.expect(t, 400, cust.order(map[int]int{available: 1, soldOut: 1})).contains(t, "ya no está disponible")
		cust.expect(t, 201, cust.order(map[int]int{available: 2}))
		cust.expect(t, 429, cust.order(map[int]int{available: 1}))
	})

	var orderID int
	scan(t, db, `SELECT id FROM orders ORDER BY id DESC LIMIT 1`, &orderID)
	orderPath := fmt.Sprintf("/order/%d", orderID)

	step(t, "order pages without htmx", func(t *testing.T) {
		cust.expect(t, 200, cust.get("/orders", false))
		cust.expect(t, 200, cust.get("/order", false)).contains(t, "Mis órdenes")
		cust.expect(t, 200, cust.get(orderPath, false)).contains(t, "Mis órdenes")
	})

	step(t, "deactivate refused", func(t *testing.T) {
		deact := func(pw string) *result {
			return cust.form("PATCH", "/account/deactivate", url.Values{"password": {pw}}, true)
		}
		cust.expect(t, 400, deact("x"))
		cust.expect(t, 401, deact("mala12345"))
		cust.expect(t, 409, deact("clave1234")).contains(t, "orden pendiente")
	})

	step(t, "staff reviews order", func(t *testing.T) {
		admin.expect(t, 200, admin.get("/dashboard/orders", true)).contains(t, fmt.Sprint(orderID))
		status := func(s int) *result {
			return admin.form("PATCH", fmt.Sprintf("/dashboard/orders/%d/status", orderID),
				url.Values{"status": {fmt.Sprint(s)}}, true)
		}
		admin.expect(t, 400, status(4))
		admin.expect(t, 200, status(3)).contains(t, "Aceptada")
		admin.expect(t, 200, status(4)).contains(t, "Cumplida")
		admin.expect(t, 200, admin.get("/dashboard/orders/history", true)).contains(t, fmt.Sprint(orderID))
	})

	step(t, "customer sees result", func(t *testing.T) {
		cust.expect(t, 200, cust.get(orderPath, true)).contains(t, "Cumplida")
	})

	step(t, "deactivate", func(t *testing.T) {
		res := cust.expect(t, 200, cust.form("PATCH", "/account/deactivate", url.Values{"password": {"clave1234"}}, true))
		if res.header.Get("HX-Redirect") != "/login" {
			t.Errorf("HX-Redirect = %q, want /login", res.header.Get("HX-Redirect"))
		}
		cust.expect(t, 200, cust.form("POST", "/login", creds("cliente", "clave1234"), false)).contains(t, "Cuenta no encontrada")
	})

	step(t, "no error logs", func(t *testing.T) {
		for _, line := range logLines(t, dir) {
			if strings.Contains(line, "level=ERROR") { t.Error(line) }
		}
	})
}

func step(t *testing.T, name string, f func(t *testing.T)) {
	if !t.Run(name, f) { t.FailNow() }
}

func initForm(key string) url.Values {
	return url.Values{
		"key": {key}, "username": {"admin"}, "full-name": {"Admin Prueba"},
		"password-1": {"admin1234"}, "password-2": {"admin1234"},
	}
}

func creds(user, pass string) url.Values {
	return url.Values{"username": {user}, "password": {pass}}
}

// --- servers ---

func build(t *testing.T, dir, tag, name string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-tags="+tag, "-o", filepath.Join(dir, name), "./cmd/main.go")
	cmd.Dir = "../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}
}

// start runs a binary on a free port and waits until it answers.
func start(t *testing.T, dir, name string) string {
	t.Helper()
	port := freePort(t)
	var stderr bytes.Buffer
	cmd := exec.Command(filepath.Join(dir, name), "-port", fmt.Sprint(port), "-quiet")
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil { t.Fatal(err) }
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second): cmd.Process.Kill(); <-done
		}
		for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
			// sonic warns that it falls back to encoding/json on this CPU.
			if line != "" && !strings.Contains(line, "sonic") { t.Errorf("%s stderr: %s", name, line) }
		}
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		if res, err := http.Get(base + "/favicon.ico"); err == nil {
			res.Body.Close()
			return base
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s did not start:\n%s", name, stderr.String())
	return ""
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func logLines(t *testing.T, dir string) []string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "log", "*.log"))
	var lines []string
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil { t.Fatal(err) }
		sc := bufio.NewScanner(fh)
		for sc.Scan() { lines = append(lines, sc.Text()) }
		fh.Close()
	}
	return lines
}

func dumpLogsOnFailure(t *testing.T, dir string) {
	if !t.Failed() { return }
	t.Log("server logs:\n" + strings.Join(logLines(t, dir), "\n"))
}

// --- database ---

func openDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(dir, "data.db")+"?_busy_timeout=5000")
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { db.Close() })
	return db
}

func seed(t *testing.T, db *sql.DB) {
	t.Helper()
	b, err := os.ReadFile("../../scripts/seed_items.sql")
	if err != nil { t.Fatal(err) }
	if _, err := db.Exec(string(b)); err != nil { t.Fatal(err) }
}

func scan(t *testing.T, db *sql.DB, query string, dst any) {
	t.Helper()
	if err := db.QueryRow(query).Scan(dst); err != nil { t.Fatalf("%s: %v", query, err) }
}

// --- HTTP ---

// client is one browser: its own cookies and User-Agent, which also gives it
// its own fingerprint.
type client struct {
	base string
	ua   string
	http *http.Client
}

type result struct {
	method, path string
	status       int
	header       http.Header
	body         string
}

func newClient(t *testing.T, base, ua string) *client {
	jar, err := cookiejar.New(nil)
	if err != nil { t.Fatal(err) }
	return &client{base: base, ua: ua, http: &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
		// Tests check redirects themselves.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (c *client) do(method, path string, body io.Reader, contentType string, hx bool) *result {
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil { panic(err) }
	req.Header.Set("User-Agent", c.ua)
	if contentType != "" { req.Header.Set("Content-Type", contentType) }
	if hx { req.Header.Set("HX-Request", "true") }
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "csrf-token" { req.Header.Set("X-CSRF-Token", ck.Value) }
	}
	res, err := c.http.Do(req)
	if err != nil { return &result{method: method, path: path, body: err.Error()} }
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return &result{method, path, res.StatusCode, res.Header, string(b)}
}

func (c *client) get(path string, hx bool) *result { return c.do("GET", path, nil, "", hx) }

func (c *client) form(method, path string, v url.Values, hx bool) *result {
	return c.do(method, path, strings.NewReader(v.Encode()), "application/x-www-form-urlencoded", hx)
}

// order posts the way cart.js does.
func (c *client) order(items map[int]int) *result {
	b, _ := json.Marshal(map[string]any{"ref_num": refNum, "items": items})
	return c.do("POST", "/order", bytes.NewReader(b), "application/json", true)
}

func (c *client) expect(t *testing.T, status int, r *result) *result {
	t.Helper()
	if r.status != status {
		t.Fatalf("%s %s: status %d, want %d\n%s", r.method, r.path, r.status, status, r.body)
	}
	return r
}

func (r *result) contains(t *testing.T, s string) {
	t.Helper()
	if !strings.Contains(r.body, s) { t.Errorf("%s %s: body lacks %q\n%s", r.method, r.path, s, r.body) }
}

func (r *result) lacks(t *testing.T, s string) {
	t.Helper()
	if strings.Contains(r.body, s) { t.Errorf("%s %s: body has %q", r.method, r.path, s) }
}
