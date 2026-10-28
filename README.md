# Repostele

Ordering system for a bakery, made of two web apps that share one SQLite
database:

- **User app** (customers): browse the menu, fill a cart, pay by SINPE and
  submit the 25-digit reference code (typed in, or read from a screenshot
  with in-browser OCR), then follow the order.
- **Staff app**: review the order queue (accept, deny, fulfill), order
  history with CSV export, dashboard stats, and admin pages for the menu,
  staff, customers and sessions.

Both are the same Go program, built with a different build tag (`USER` or
`STAFF`). Pages are server-rendered with `html/template` and updated with
htmx. Everything the browser needs is served by the apps themselves, so the
system runs on a network with no Internet access.

## Requirements

- Go 1.26+ and a C compiler (`github.com/mattn/go-sqlite3` uses cgo)
- [Dart Sass](https://sass-lang.com/dart-sass/) (`sass`) to build the CSS
- `sqlite3` command-line tool, for seeding and backups

## Build

```sh
make all      # CSS, then bin/user and bin/staff
make vet      # go vet for both build tags
make test     # end-to-end test (see Testing)
```

Do not run `make clean` casually: it deletes the binaries and the generated
CSS (the database is kept).

## Run

```sh
scripts/start.sh   # staff on :8081, user on :8080, in the background
scripts/stop.sh
```

Ports can be changed with `STAFF_PORT` and `USER_PORT`. The staff app creates
`bin/data.db` from `static/schema.sql` the first time it runs; the user app
never creates it, so `start.sh` always starts staff first and, on a fresh
install, waits for it before starting the user app.

Each binary can also be run by hand from anywhere; it changes to its own
directory first, so `data.db`, `dyn/` and `log/` always live next to it:

```sh
bin/staff -port 8081           # logs to bin/log and stdout
bin/user  -port 8080 -quiet    # log file only
```

### First start

1. Open the staff app (`http://<host>:8081`). With no staff accounts yet,
   every page leads to `/init`.
2. Enter the activation key and the first admin's details. The key is
   `_TEST_KEY` in `internal/controllers/staff.go`.
3. Optionally load sample menu items:
   `sqlite3 bin/data.db < scripts/seed_items.sql` (safe to re-run).

Customers register themselves in the user app (one account per device).
Further staff accounts are created by an admin from the dashboard.

## Data, logs and backups

Everything a running system writes is under `bin/`:

| Path | Contents |
|------|----------|
| `bin/data.db` (+ `-wal`, `-shm`) | the database, in WAL mode |
| `bin/dyn/img/` | uploaded menu images |
| `bin/log/<app>.<timestamp>.log` | one log file per server start |
| `bin/log/<app>.err` | startup failures when run by `start.sh` |

Recent writes live in `data.db-wal` until SQLite folds them into `data.db`,
so never copy `data.db` alone. Back up with:

```sh
scripts/backup.sh            # into bin/backup/<timestamp>/
scripts/backup.sh /mnt/usb   # or into another directory
```

It works while the servers are running and copies the database and the
uploaded images.

To restore a backup:

```sh
scripts/stop.sh
rm -f bin/data.db-wal bin/data.db-shm
cp bin/backup/<timestamp>/data.db bin/data.db
rm -rf bin/dyn && cp -r bin/backup/<timestamp>/dyn bin/dyn
scripts/start.sh
```

Logs are text (logfmt), one line per request plus error lines tied to it:

```
level=INFO msg="POST /order => HTTP 201 (336µs)" bin=user req=10 user=cliente
level=ERROR msg="..." bin=user method=POST path=/order req=11 user=cliente
```

`req` is a per-process request number that links an error to its request
line. Notification polls with nothing new are not logged.

## Testing

`make test` builds both apps into a temporary directory, starts them on free
ports against a fresh database and walks the main flows over HTTP: setup,
registration, ordering, staff review, account deactivation. It also fails if
the servers wrote any `ERROR` log line. It never touches `bin/`.

Steps build on each other (one pair of servers for the whole run), so the
first failing step stops the test, and its output ends with the server logs.

OCR runs in the browser and is not covered by this test.

## Layout

```
cmd/main.go            entry point: flags, logging, router
internal/routes        URL routes; build_user.go / build_staff.go per app
internal/controllers   HTTP handlers and page rendering
internal/models        database access (sqlx)
internal/mymiddleware  sessions, CSRF, fingerprints, init check
internal/reqlog        request numbers and request-tied error logs
pkg/bind               form/JSON binding and validation
pkg/pass               password hashing and tokens
pkg/upload             image uploads
static/schema.sql      database schema
static/html            page templates (main-<section>.gohtml)
static/htmx            partial templates for htmx swaps
static/public          files served as-is: css/, js/, img/, vendor/
scss/                  stylesheet sources (built into static/public/css)
scripts/               start, stop, backup, seed
test/e2e               end-to-end test
```

Files named `build_user.go` / `build_staff.go` hold the parts that differ
between the two apps; everything else is shared. Templates and
`static/public` are embedded in the binaries.

### Vendored browser libraries

| Library | Version | Path |
|---------|---------|------|
| htmx | 2.0.10 | `static/public/vendor/htmx.min.js` |
| tesseract.js | 6.0.1 (core 6.0.0) | `static/public/vendor/tesseract/` |
| English OCR data | 4.0.0_best_int | `static/public/vendor/tesseract/eng.traineddata.gz` |
