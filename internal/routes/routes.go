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
  r.Handle("/*", cacheStatic(http.FileServerFS(sub)))
  r.Handle(
    "/"+_DYNDIR+"/*",
    cacheFor(_CACHE_LONG, http.StripPrefix(
      "/"+_DYNDIR,
      http.FileServer(http.Dir(_DYNDIR)),
    )),
  )
  r.Get("/htmx/notif", controllers.ServeNotif)
  r.Get("/"+static.HXDIR+"/{path}", controllers.ServeHTMX)
  return nil
}

const (
  _CACHE_LONG  = "public, max-age=31536000, immutable"
  // Embedded files have no modtime to revalidate against; unversioned ones
  // (CSS background images) get a short lifetime instead.
  _CACHE_SHORT = "public, max-age=3600"
)

func cacheFor(policy string, h http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", policy)
    h.ServeHTTP(w, r)
  })
}

// cacheStatic: URLs from the asset template func carry ?v=<hash> and never change.
func cacheStatic(h http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    policy := _CACHE_SHORT
    if r.URL.Query().Has("v") { policy = _CACHE_LONG }
    w.Header().Set("Cache-Control", policy)
    h.ServeHTTP(w, r)
  })
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
