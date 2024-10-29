package repository

import "github.com/gofiber/fiber/v2"

func (repo *Repository) SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/paket", JWTMiddleware, repo.GetPaket)
	api.Post("/paket", JWTMiddleware, repo.CreatePaket)
	api.Patch("/paket/:id", repo.UpdatePaket)
	api.Delete("/paket/:id", repo.DeletePaket)
	api.Get("/paket/:id", repo.GetPaketByID)
	api.Patch("/paket/done/:id", repo.MarkPaketAsDone)
	// login and register
	api.Post("/register", repo.Register)
	api.Post("/login", repo.Login)
}
