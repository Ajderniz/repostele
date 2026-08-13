package routes

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/controllers"
	"github.com/ajderniz/repostele/internal/mymiddleware"
	"github.com/ajderniz/repostele/static"
)

const _DYNDIR = "dyn"

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
  r.Get("/"+static.HXDIR+"/{path}", controllers.ServeHTMX)
  return nil
}

func RegisterUserRoutes(r *chi.Mux) error {

  err := setupFileServer(r)
  if err != nil { return err }
  
  r.Get("/", controllers.HandleRootUser)

  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitUser())
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Route("/", func(r chi.Router){
    r.Use(mymiddleware.CheckInitUser())
    r.Use(mymiddleware.GetFingerprint())
    r.Post("/register", controllers.RegisterUserAccount)
    r.Post("/login",    controllers.UserLogin)
  })
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitUser())
    r.Use(mymiddleware.RequireUserAuth())
    r.Patch("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitUser())
    r.Use(mymiddleware.RequireUserAuth())
    r.Patch("/deactivate", controllers.SelfDeactivateUserAccount)
    r.Patch("/password",   controllers.SelfUpdateUserPassword)
  })

  r.Route("/order", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitUser())
    r.Use(mymiddleware.RequireUserAuth())
    r.Post( "/",       controllers.PostOrder)
    r.Get(  "/",       controllers.GetUserOrderList)
    r.Get(  "/{id}",   controllers.CheckUserOrderFromID)
    r.Patch("/update", controllers.UpdateUserOrderRefNum)
    r.Patch("/cancel", controllers.CancelUserOrder)
  })

  return nil
}

func RegisterStaffRoutes(r *chi.Mux) error {

  err := setupFileServer(r)
  if err != nil { return err }

  r.Get( "/",     controllers.HandleRootStaff)
  r.Route("/init", func(r chi.Router){
    r.Get( "/",     controllers.ServeTemplateStaff)
    r.Post("/",     controllers.InitMainStaffAccount)
  })

  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitStaff())
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Route("/login", func(r chi.Router){
    r.Use(mymiddleware.CheckInitStaff())
    r.Use(mymiddleware.GetFingerprint())
    r.Post("/", controllers.StaffLogin)
  })
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitStaff())
    r.Use(mymiddleware.RequireStaffAuth())
    r.Patch("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitStaff())
    r.Use(mymiddleware.RequireStaffAuth())
    r.Patch("/deactivate", controllers.SelfDeactivateStaffAccount)
    r.Patch("/password",   controllers.SelfUpdateStaffPassword)
  })

  r.Route("/dashboard", func(r chi.Router) {
    r.Use(mymiddleware.CheckInitStaff())
    r.Use(mymiddleware.RequireStaffAuth())

    r.Route("/orders", func(r chi.Router) {
      r.Get(  "/",            controllers.GetAllOrders)
      r.Get(  "/{id}",        controllers.GetOrderFromID)
      r.Patch("/{id}/status", controllers.UpdateOrderStatus)
    })

    r.Route("/admin", func(r chi.Router) {
      r.Use(mymiddleware.RequireAdminAuth())

      r.Route("/staff", func(r chi.Router) {
        r.Get(  "/",                      controllers.GetStaffList)
        r.Post( "/",                      controllers.RegisterStaffAccount)
        r.Get(  "/{username}",            controllers.GetStaffFromUsername)
        r.Patch("/{username}/password",   controllers.UpdateStaffPassword)
        r.Patch("/deactivate/{username}", controllers.DeactivateStaffAccount)
      })

      r.Route("/users", func(r chi.Router) {
        r.Get(  "/",                      controllers.GetUserList)
        r.Get(  "/{username}",            controllers.GetUserFromUsername)
        r.Patch("/{username}/password",   controllers.UpdateUserPassword)
        r.Patch("/deactivate/{username}", controllers.DeactivateUserAccount)
      })

      r.Route("/sessions", func(r chi.Router) {
        r.Get(  "/",                 controllers.GetActiveSessions)
        r.Patch("/close",            controllers.CloseAllSessions)
        r.Patch("/close/{username}", controllers.CloseSessionForUsername)
      })

      r.Route("/menu", func(r chi.Router) {
        r.Post( "/",     controllers.PostItem)
        r.Patch("/{id}", controllers.UpdateItem)
      })
    })
  })

  return nil
}