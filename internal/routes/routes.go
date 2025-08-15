package routes

import (
	"storage-api/internal/handlers"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

func SetupRoutes(app *fiber.App) {
	// API routes group
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Monitor route
	app.Get("/metrics", monitor.New())

	// Health check route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"service":   "storage-api",
			"timestamp": time.Now().UTC(),
		})
	})

	// File routes
	fileHandler := handlers.NewFileHandler()

	files := v1.Group("/files")
	files.Post("/", fileHandler.UploadFile)
	files.Get("/", fileHandler.SearchFiles)
	files.Get("/limits", fileHandler.GetFileLimits)
	files.Get("/:id", fileHandler.GetFile)
	files.Get("/:id/download", fileHandler.DownloadFile)
	files.Put("/:id", fileHandler.UpdateFile)
	files.Delete("/:id", fileHandler.DeleteFile)
}
