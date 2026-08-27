package routes

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

  "github.com/ajderniz/repostele/internal/controllers"
  "github.com/ajderniz/repostele/static"
)

const _DYNDIR = static.DYNDIR

func setupFileServer(r *chi.Mux) error {
  sub, err := fs.Sub(static.FS, static.PUBLIC)
  if err != nil { return err }
  r.Handle("/*", http.FileServerFS(sub))
  r.Handle(
    "/"+_DYNDIR+"/*",
    http.StripPrefix(
      "/"+_DYNDIR,
      http.FileServer(http.Dir(_DYNDIR)),
    ),
  )
  r.Get("/htmx/nav", controllers.ServeNav)
  r.Get("/"+static.HXDIR+"/{path}", controllers.ServeHTMX)
  return nil
}
