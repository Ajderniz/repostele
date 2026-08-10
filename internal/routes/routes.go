package routes

import (
  "github.com/go-chi/chi/v5"

  "github.com/ajderniz/repostele/internal/controllers"
  "github.com/ajderniz/repostele/internal/mymiddleware"
)

func RegisterUserRoutes(r *chi.Mux) {

  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Route("/", func(r chi.Router){
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.GetFingerprint())
    r.Post("/register", controllers.RegisterUserAccount)
    r.Post("/login",    controllers.UserLogin)
  })
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireUserAuth())
    r.Put("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireUserAuth())
    r.Put("/deactivate", controllers.SelfDeactivateUserAccount)
    r.Put("/password",   controllers.SelfUpdateUserPassword)
  })

  r.Route("/order", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireUserAuth())
    r.Post("/",       controllers.PostOrder)
    r.Get( "/",       controllers.GetUserOrderList)
    r.Get( "/{id}",   controllers.CheckUserOrderFromID)
    r.Put( "/update", controllers.UpdateUserOrderRefNum)
    r.Put( "/cancel", controllers.CancelUserOrder)
  })
}

func RegisterStaffRoutes(r *chi.Mux) {
  r.Post("/init",  controllers.InitMainStaffAccount)

  r.Route("/menu", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Get("/",          controllers.GetItems)
    r.Get("/item/{id}", controllers.GetItemFromID)
  })

  r.Route("/login", func(r chi.Router){
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.GetFingerprint())
    r.Post("/", controllers.StaffLogin)
  })
  r.Route("/logout", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireStaffAuth())
    r.Put("/", controllers.Logout)
  })

  r.Route("/account", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
    r.Use(mymiddleware.RequireStaffAuth())
    r.Put("/deactivate", controllers.SelfDeactivateStaffAccount)
    r.Put("/password",   controllers.SelfUpdateStaffPassword)
  })

  r.Route("/dashboard", func(r chi.Router) {
    r.Use(mymiddleware.CheckInit())
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
        r.Post("/",                      controllers.RegisterStaffAccount)
        r.Get( "/{username}",            controllers.GetStaffFromUsername)
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
        r.Get("/",                 controllers.GetActiveSessions)
        r.Put("/close",            controllers.CloseAllSessions)
        r.Put("/close/{username}", controllers.CloseSessionForUsername)
      })

      r.Route("/menu", func(r chi.Router) {
        r.Post("/",     controllers.PostItem)
        r.Put( "/{id}", controllers.UpdateItem)
      })
    })
  })
}