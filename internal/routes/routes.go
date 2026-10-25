package routes

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

  "github.com/ajderniz/repostele/internal/controllers"
  "github.com/ajderniz/repostele/internal/mymiddleware"
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
  r.Get("/htmx/notif", controllers.ServeNotif)
  r.Get("/"+static.HXDIR+"/{path}", controllers.ServeHTMX)
  return nil
}

// servePage mounts a section whose GET just renders its main template.
func servePage(r chi.Router) {
  r.Get("/", controllers.ServeMainTemplate)
}

// registerShared adds the routes both apps have; the caller applies CheckInit.
func registerShared(r chi.Router) {
  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.OptUsername())
    r.Get("/", controllers.GetItems)
  })

  r.Route("/login", func(r chi.Router){
    r.Use(mymiddleware.GetFingerprint())
    r.Use(mymiddleware.OptUsername())
    r.Get( "/", controllers.ServeMainTemplate)
    r.Post("/", controllers.Login)
  })

  r.Group(func(r chi.Router) {
    r.Use(mymiddleware.RequireAuth())

    r.Route("/logout", func(r chi.Router) {
      r.Patch("/", controllers.Logout)
    })

    r.Route("/account", func(r chi.Router) {
      r.Get(  "/",           controllers.ServeMainTemplate)
      r.Patch("/deactivate", controllers.SelfDeactivateAccount)
      r.Patch("/password",   controllers.SelfUpdatePassword)
    })
  })
}
