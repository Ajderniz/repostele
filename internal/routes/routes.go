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

  r.Post("/register", controllers.RegisterUserAccount)
  r.Post("/login",    controllers.UserLogin)
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.RequireUserAuth())
    r.Post("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.RequireUserAuth())
    r.Put("/deactivate",      controllers.DeactivateUserAccountSelf)
    r.Put("/update/username", controllers.UpdateUserUsername)
    r.Put("/update/password", controllers.UpdateUserPasswordSelf)
  })

  r.Route("/order", func(r chi.Router) {
    r.Use(mymiddleware.RequireUserAuth())
    r.Post("/",       controllers.PostOrder)
    r.Get( "/",       controllers.GetUserOrderList)
    r.Get( "/{id}",   controllers.CheckUserOrderFromID)
    r.Put( "/update", controllers.UpdateUserOrderRefNum)
    r.Put( "/cancel", controllers.CancelUserOrder)
  })
}

func RegisterStaffRoutes(r *chi.Mux) {
  r.Route("/menu", func(r chi.Router) {
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Post("/init",  controllers.RegisterStaffAccountInit)
  r.Post("/login", controllers.StaffLogin)
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.RequireStaffAuth())
    r.Post("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.RequireStaffAuth())
    r.Put("/deactivate", controllers.DeactivateStaffAccountSelf)
    r.Put("/username",   controllers.UpdateStaffUsername)
    r.Put("/password",   controllers.UpdateStaffPasswordSelf)
  })

  r.Route("/dashboard", func(r chi.Router) {
    r.Use(mymiddleware.RequireStaffAuth())

    r.Route("/orders", func(r chi.Router) {
      r.Get("/",            controllers.GetAllOrders)
      r.Get("/{id}",        controllers.GetOrderFromID)
      r.Put("/{id}/status", controllers.UpdateOrderStatus)
    })

    r.Route("/admin", func(r chi.Router) {
      r.Use(mymiddleware.RequireAdminAuth())

      r.Route("/staff", func(r chi.Router) {
        r.Get( "/",                      controllers.GetStaffList)
        r.Get( "/{username}",            controllers.GetStaffFromUsername)
        r.Post("/register",              controllers.RegisterStaffAccount)
        r.Put( "/{username}/password",   controllers.UpdateStaffPassword)
        r.Put( "/deactivate/{username}", controllers.DeactivateStaffAccount)
      })

      r.Route("/users", func(r chi.Router) {
        r.Get("/",                      controllers.GetUserList)
        r.Get("/{username}",            controllers.GetUserFromUsername)
        r.Put("/{username}/password",   controllers.UpdateUserPassword)
        r.Put("/deactivate/{username}", controllers.DeactivateUserAccount)
      })

      r.Route("/sessions", func(r chi.Router) {
        r.Get("/",                   controllers.GetSessions)
        r.Put("/close",              controllers.CloseAllSessions)
        r.Put("/close/{session_id}", controllers.CloseSessionWithID)
      })

      r.Route("/menu", func(r chi.Router) {
        r.Post("/",     controllers.PostItem)
        r.Put( "/{id}", controllers.UpdateItem)
      })
    })
  })
}