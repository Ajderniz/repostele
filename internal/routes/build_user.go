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

  r.Group(func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    registerShared(r)

    r.Route("/register", func(r chi.Router){
      r.Use(mymiddleware.GetFingerprint())
      r.Use(mymiddleware.OptUsername())
      r.Post("/", controllers.SelfRegisterAccount)
    })

    r.Group(func(r chi.Router) {
      r.Use(mymiddleware.RequireAuth())

      r.Route("/cart", func(r chi.Router) {
        r.Get("/", controllers.GetCartPage)
      })
      r.Route("/orders", servePage)

      r.Route("/order", func(r chi.Router) {
        r.Post( "/",          controllers.PostOrder)
        r.Get(  "/",          controllers.GetUserOrderList)
        r.Get(  "/edit-form", controllers.GetOrderRefNumEditForm)
        r.Get(  "/ref-view",  controllers.GetOrderRefView)
        r.Get(  "/{id}",      controllers.CheckUserOrderFromID)
        r.Patch("/update",    controllers.UpdateUserOrderRefNum)
        r.Patch("/cancel",    controllers.CancelUserOrder)
      })
    })
  })

  return nil
}
