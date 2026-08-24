//go:build STAFF

package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/controllers"
	"github.com/ajderniz/repostele/internal/mymiddleware"
)

func RegisterRoutes(r *chi.Mux) error {

  err := setupFileServer(r)
  if err != nil { return err }

  r.Get( "/",     controllers.HandleRoot)
  r.Route("/init", func(r chi.Router){
    r.Get( "/",     controllers.ServeMainTemplate)
    r.Post("/",     controllers.InitMainStaffAccount)
  })

  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.OptUsername())
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Route("/login", func(r chi.Router){
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.GetFingerprint())
    r.Use(mymiddleware.OptUsername())
    r.Get( "/", controllers.ServeMainTemplate)
    r.Post("/", controllers.Login)
  })
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireAuth())
    r.Patch("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireAuth())
    r.Get(  "/",           controllers.ServeMainTemplate)
    r.Patch("/deactivate", controllers.SelfDeactivateAccount)
    r.Patch("/password",   controllers.SelfUpdatePassword)
  })

  r.Route("/dashboard", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireAuth())

    r.Get("/", controllers.ServeMainTemplate)

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
