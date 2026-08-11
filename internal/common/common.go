package common

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	port := flag.Int("port", 8080, "Port number")
	flag.Parse()

	exePath, err := os.Executable()
	if err != nil { return err }
	exeDir := filepath.Dir(exePath)
	err = os.Chdir(exeDir)
	if err != nil { return err }

	exeBase := filepath.Base(exePath)
	logFile, err := os.OpenFile(
		exeBase+"."+time.Now().Format("060102_150405")+".log",
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0644,
	)
	if err != nil { return err }
	logger := slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stdout, nil),
		slog.NewTextHandler(logFile, nil),
	))
	slog.SetDefault(logger)

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

	if msg != "" { slog.Info(msg) }
	slog.Info("Listening", slog.Int("port", *port))

	return http.ListenAndServe(":" + strconv.Itoa(*port), r)
}