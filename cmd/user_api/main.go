package main

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
	"github.com/ajderniz/repostele/pkg/errman"
)

func main() {
	port := flag.Int("port", 8080, "Port number")
	flag.Parse()

	exe, err := os.Executable()
	errman.CheckFatal(err)
	exeDir := filepath.Dir(exe)
	errman.CheckFatal(err)
	err = os.Chdir(exeDir)
	errman.CheckFatal(err)

	err = models.OpenDB()

	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.CleanPath,
		middleware.AllowContentEncoding("application/json"),
		middleware.Throttle(20))
	routes.Register(r)

	fmt.Println("Listening on port ", *port)
	err = http.ListenAndServe(":" + strconv.Itoa(*port), r)
	errman.CheckFatal(err)
}