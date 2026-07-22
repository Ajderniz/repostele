package routes

import (
  "github.com/go-chi/chi/v5"

  "github.com/ajderniz/repostele/internal/controllers"
  "github.com/ajderniz/repostele/internal/mymiddleware"
)

func RegisterUserRoutes(r *chi.Mux) {
  r.Route("/menu", func(r chi.Router) {
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Post("/account/register", controllers.RegisterUserAccount)
  r.Post("/account/login",    controllers.UserLogin)
  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.RequireUserAuth())
    r.Delete("/deactivate", controllers.DeactivateUserAccount)
    r.Post("/logout", controllers.UserLogout)
  })

  r.Route("/order", func(r chi.Router) {
    r.Use(mymiddleware.RequireUserAuth())
    r.Post("/",       controllers.PostOrder)
    r.Get( "/",       controllers.GetOrderList)
    r.Get( "/{id}",   controllers.CheckOrderFromID)
    r.Put( "/update", controllers.UpdateOrderRefNum)
    r.Put( "/cancel", controllers.CancelOrder)
  })
}

func RegisterStaffRoutes(r *chi.Mux) {
  r.Post("/account/register", controllers.RegisterStaffAccount)
  r.Post("/account/login", controllers.StaffLogin)
  r.Route("/dashboard", func(r chi.Router) {
    r.Use(mymiddleware.RequireStaffAuth())
  })
}