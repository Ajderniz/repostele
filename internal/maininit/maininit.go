package maininit

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
	"github.com/go-chi/httplog/v3"

	"github.com/ajderniz/repostele/internal/controllers"
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

	controllers.ParseTemplates()

	err = models.OpenDB()
	if err != nil { return err }

	r := chi.NewRouter()

	logFormat := httplog.SchemaECS.Concise(true)
	r.Use(
		httplog.RequestLogger(
			logger,
			&httplog.Options{
				Level: slog.LevelInfo,
				Schema: logFormat,
				RecoverPanics: true,
			},
		),
		middleware.CleanPath,
		middleware.AllowContentEncoding("application/json"),
		middleware.Throttle(20),
	)
	
	if bin == SERVER_BINARY_STAFF {
	  err = routes.RegisterStaffRoutes(r)
	} else {
	  err = routes.RegisterUserRoutes(r)
	}
	if err != nil { return err }

	if msg != "" { slog.Info(msg) }
	slog.Info("Listening", slog.Int("port", *port))

	return http.ListenAndServe(":" + strconv.Itoa(*port), r)
}