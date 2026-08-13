package main

import (
	"flag"
	"log"
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
	"github.com/ajderniz/repostele/internal/mainbuild"
	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/internal/routes"
)

const _LOG_DIR = "log"

func main() {
	port := flag.Int("port", 8080, "Port number")
	flag.Parse()

	exePath, err := os.Executable()
	if err != nil { log.Fatal(err.Error()) }
	exeDir := filepath.Dir(exePath)
	err = os.Chdir(exeDir)
	if err != nil { log.Fatal(err.Error()) }

	os.MkdirAll(_LOG_DIR, os.ModePerm)
	exeBase := filepath.Base(exePath)
	logFile, err := os.OpenFile(
		_LOG_DIR+"/"+exeBase+"."+time.Now().Format("060102150405")+".log",
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0644,
	)
	if err != nil { log.Fatal(err.Error()) }

	logger := slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stdout, nil),
		slog.NewTextHandler(logFile, nil),
	))
	slog.SetDefault(logger)

	err = models.OpenDB()
	if err != nil { log.Fatal(err.Error()) }

	r := chi.NewRouter()

	chiLogFormat := httplog.SchemaECS.Concise(true)
	r.Use(
		httplog.RequestLogger(
			logger,
			&httplog.Options{
				Level: slog.LevelInfo,
				Schema: chiLogFormat,
				RecoverPanics: true,
			},
		),
		middleware.CleanPath,
		middleware.AllowContentEncoding("application/json"),
		middleware.Throttle(20),
	)

	controllers.InitTemplate()

  err = routes.RegisterRoutes(r)
	if err != nil { log.Fatal(err.Error()) }

	slog.Info("Repostele: "+mainbuild.INIT_MSG)
	slog.Info("Listening", slog.Int("port", *port))

	err = http.ListenAndServe(":" + strconv.Itoa(*port), r)
	if err != nil { log.Fatal(err.Error()) }
}
