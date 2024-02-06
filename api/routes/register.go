package routes

import (
	"github.com/betterde/orbit/internal/response"
	"github.com/betterde/orbit/spa"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func RegisterRoutes(app *fiber.App) {
	app.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.JSON(response.Success("Success", nil, nil))
	}).Name("Health check")

	api := app.Group("/api")

	api.Get("users", func(ctx *fiber.Ctx) error {
		return ctx.JSON(response.Success("Success", nil, nil))
	}).Name("Query users list")

	// Embed SPA static resource
	app.All("/*", filesystem.New(filesystem.Config{
		Root:               spa.Serve(),
		Index:              "index.html",
		NotFoundFile:       "index.html",
		ContentTypeCharset: "UTF-8",
	}))
}
