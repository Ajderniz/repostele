package routes

import (
  "github.com/go-chi/chi/v5"

  "github.com/ajderniz/repostele/internal/controllers"
  "github.com/ajderniz/repostele/internal/mymiddleware"
)

func Register(r *chi.Mux) {
  r.Route("/menu", func(r chi.Router) {
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Post("/register", controllers.RegisterUser)
  r.Post("/login",    controllers.UserLogin)
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.RequireAuth())
    r.Post("/", controllers.UserLogout)
  })
  r.Route("/user", func(r chi.Router) {
    r.Use(mymiddleware.RequireAuth())
    r.Delete("/delete", controllers.DeleteUser)
  })

  r.Route("/order", func(r chi.Router) {
    r.Use(mymiddleware.RequireAuth())
    r.Post("/",       controllers.PostOrder)
    r.Get( "/",       controllers.GetOrderList)
    r.Get( "/{id}",   controllers.CheckOrderFromID)
    r.Put( "/update", controllers.UpdateOrderRefNum)
    r.Put( "/cancel", controllers.CancelOrder)
  })
}