package controllers

import (
	"fmt"
	"hyperpage/initializers"
	"hyperpage/models"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/satori/go.uuid"
)

var allowedMIMETypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/gif":          true,
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"audio/mpeg":       true,
	"audio/wav":        true,
	"video/mp4":        true,
	"video/x-matroska": true,
	// Добавьте другие допустимые MIME-типы при необходимости
}

func CreatePost(c *fiber.Ctx) error {
	post := new(models.Post)

	// Парсинг JSON данных (текст поста и др.)
	if err := c.BodyParser(post); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	post.ID = uuid.NewV4() // Генерация нового UUID для поста

	// Получение информации о пользователе из контекста
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}
	post.UserID = userResponse.ID

	// Обработка прикрепленных файлов, если они есть
	form, err := c.MultipartForm()
	if err != nil && err != fiber.ErrUnprocessableEntity {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get multipart form",
		})
	}

	if form != nil {
		files := form.File["files"]
		for _, file := range files {
			// Проверка MIME-типа файла
			if !allowedMIMETypes[file.Header.Get("Content-Type")] {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("File type %s is not allowed", file.Header.Get("Content-Type")),
				})
			}

			// Генерация пути для сохранения файла
			fileID := uuid.NewV4()
			filePath := filepath.Join("uploads", fmt.Sprintf("%s-%s", fileID.String(), file.Filename))

			// Сохранение файла на сервере
			if err := c.SaveFile(file, filePath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to save file",
				})
			}

			// Создание записи о файле в базе данных
			fileRecord := models.FilePost{
				ID:     fileID,
				URL:    filePath,
				PostID: post.ID,
			}
			post.Files = append(post.Files, fileRecord)
		}
	}

	// Сохранение поста в базе данных
	if err := initializers.DB.Create(&post).Error; err != nil {
		// Удаление сохраненных файлов, если пост не был сохранен
		for _, file := range post.Files {
			os.Remove(file.URL)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot create post",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(post)
}
