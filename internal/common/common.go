package common

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/internal/routes"
)

type ServerBinary int
const (
	SERVER_BINARY_STAFF ServerBinary = iota
	SERVER_BINARY_USER
)

func InitMain(msg string, bin ServerBinary) error {
	port := *flag.Int("port", 8080, "Port number")
	flag.Parse()

	exe, err := os.Executable()
	if err != nil { return err }
	exeDir := filepath.Dir(exe)
	err = os.Chdir(exeDir)
	if err != nil { return err }

	err = models.OpenDB()
	if err != nil { return err }

	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.CleanPath,
		middleware.AllowContentEncoding("application/json"),
		middleware.Throttle(20),
	)
	if bin == SERVER_BINARY_STAFF {
	  routes.RegisterStaffRoutes(r)
	} else {
	  routes.RegisterUserRoutes(r)
	}

	if msg != "" { fmt.Println(msg) }
	fmt.Println("Listening on port ", port)

	return http.ListenAndServe(":" + strconv.Itoa(port), r)
}