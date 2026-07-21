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

func InitMain(msg string) error {
	port := flag.Int("port", 8080, "Port number")
	flag.Parse()

	exe, err := os.Executable()
	if err != nil { return err }
	exeDir := filepath.Dir(exe)
	if err != nil { return err }
	err = os.Chdir(exeDir)
	if err != nil { return err }

	err = models.OpenDB()

	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.CleanPath,
		middleware.AllowContentEncoding("application/json"),
		middleware.Throttle(20),
	)
	routes.Register(r)

	if msg != "" { fmt.Println(msg) }
	fmt.Println("Listening on port ", *port)

	return http.ListenAndServe(":" + strconv.Itoa(*port), r)
}