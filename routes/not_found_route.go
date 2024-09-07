package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// NotFoundRoute func for describe 404 Error route.

func NotFoundRoute(app *fiber.App) {
    app.Use(func(c *fiber.Ctx) error {
        // Проверяем, относится ли запрос к статическим файлам
        if strings.HasPrefix(c.Path(), "/s/") || strings.HasPrefix(c.Path(), "/public/") {
            return c.Next() // Пропускаем обработку, если это статика
        }

        // Если это не статика, возвращаем 404
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": true,
            "msg":   "sorry, endpoint is not found",
        })
    })
}