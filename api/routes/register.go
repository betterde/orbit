package routes

import (
	"github.com/betterde/orbit/internal/response"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App) {
	app.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.JSON(response.Success("Success", nil, nil))
	}).Name("Health check")
}
