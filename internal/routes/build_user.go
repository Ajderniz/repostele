//go:build USER

package routes

import (
	"github.com/go-chi/chi/v5"

  "github.com/ajderniz/repostele/internal/controllers"
  "github.com/ajderniz/repostele/internal/mymiddleware"
)

func RegisterRoutes(r *chi.Mux) error {

  err := setupFileServer(r)
  if err != nil { return err }

  r.Get("/", controllers.HandleRoot)

  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.OptUsername())
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Route("/register", func(r chi.Router){
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.GetFingerprint())
    r.Use(mymiddleware.OptUsername())
    r.Post("/", controllers.SelfRegisterAccount)
  })
  r.Route("/login", func(r chi.Router){
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.GetFingerprint())
    r.Use(mymiddleware.OptUsername())
    r.Get( "/",    controllers.ServeMainTemplate)
    r.Post("/",    controllers.UserLogin)
  })
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireAuth())
    r.Patch("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireAuth())
    r.Patch("/deactivate", controllers.SelfDeactivateAccount)
    r.Patch("/password",   controllers.SelfUpdatePassword)
  })

  r.Route("/order", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireAuth())
    r.Post( "/",       controllers.PostOrder)
    r.Get(  "/",       controllers.GetUserOrderList)
    r.Get(  "/{id}",   controllers.CheckUserOrderFromID)
    r.Patch("/update", controllers.UpdateUserOrderRefNum)
    r.Patch("/cancel", controllers.CancelUserOrder)
  })

  return nil
}
