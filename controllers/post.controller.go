package controllers

import (
	"fmt"
	"hyperpage/initializers"
	"hyperpage/models"
	"hyperpage/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	"video/quicktime":  true, // Добавление поддержки формата MOV
}

func CreatePost(c *fiber.Ctx) error {
	post := new(models.Post)

	if err := c.BodyParser(post); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	post.ID = uuid.NewV4()
	config, _ := initializers.LoadConfig(".")

	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		log.Printf("Error retrieving user information from context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}
	post.UserID = userResponse.ID
	post.Content = c.FormValue("content")

	// Обработка связи с блогом (если передан blog_id)
	blogID := c.FormValue("blog_id")
	if blogID != "" {
		var blog models.Blog
		if err := initializers.DB.First(&blog, "id = ?", blogID).Error; err != nil {
			log.Printf("Blog with id %s not found: %v", blogID, err)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Blog not found",
			})
		}
		post.BlogID = &blog.ID
	}

	// Обработка тегов
	tagNames := c.FormValue("tags")
	tags := []models.Tag{}
	if tagNames != "" {
		tagNamesArray := strings.Split(tagNames, ",")
		existingTags := []models.Tag{}
		initializers.DB.Where("name IN ?", tagNamesArray).Find(&existingTags)
		existingTagMap := make(map[string]models.Tag)
		for _, tag := range existingTags {
			existingTagMap[tag.Name] = tag
		}

		for _, tagName := range tagNamesArray {
			tagName = strings.TrimSpace(tagName)
			if len(tagName) == 0 || len(tagName) > 100 {
				continue // Пропуск пустых и слишком длинных тегов
			}
			if tag, found := existingTagMap[tagName]; found {
				tags = append(tags, tag)
			} else {
				newTag := models.Tag{
					ID:   uuid.NewV4(),
					Name: tagName,
				}
				tags = append(tags, newTag)
				if err := initializers.DB.Create(&newTag).Error; err != nil {
					log.Printf("Error creating tag %s: %v", tagName, err)
				}
			}
		}
	}
	post.Tags = tags

	dirPath := filepath.Join(config.IMGStorePath, userResponse.Storage)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		log.Printf("Error creating user directory: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user directory",
		})
	}

	dirSize, err := calculateDirSize(dirPath)
	if err != nil {
		log.Printf("Error calculating directory size: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to calculate directory size",
		})
	}

	if dirSize > float64(userResponse.LimitStorage) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Storage limit exceeded",
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("Error getting multipart form: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get multipart form",
		})
	}

	files := form.File["files"]
	maxFileSize := int64(10 * 1024 * 1024)
	for _, file := range files {
		if !allowedMIMETypes[file.Header.Get("Content-Type")] {
			log.Printf("File type %s is not allowed", file.Header.Get("Content-Type"))
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("File type %s is not allowed", file.Header.Get("Content-Type")),
			})
		}

		if file.Size > maxFileSize {
			log.Printf("File %s exceeds the 10MB size limit", file.Filename)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("File %s exceeds the 10MB size limit", file.Filename),
			})
		}

		fileID := uuid.NewV4()
		originalFilePath := filepath.Join(dirPath, fmt.Sprintf("%s-%s", fileID.String(), file.Filename))
		finalFilePath := originalFilePath

		if err := c.SaveFile(file, originalFilePath); err != nil {
			log.Printf("Error saving file %s: %v", file.Filename, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save file",
			})
		}

		// Конвертация MOV в MP4
		if file.Header.Get("Content-Type") == "video/quicktime" {
			finalFilePath = filepath.Join(dirPath, fmt.Sprintf("%s.mp4", fileID.String()))
			cmd := exec.Command("ffmpeg", "-i", originalFilePath, finalFilePath)
			if err := cmd.Run(); err != nil {
				log.Printf("Error converting MOV to MP4: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to convert MOV to MP4",
				})
			}
			// Удаляем оригинальный файл MOV
			if err := os.Remove(originalFilePath); err != nil {
				log.Printf("Error deleting original MOV file %s: %v", originalFilePath, err)
			}
		}

		// Конвертация WAV в MP3
		if file.Header.Get("Content-Type") == "audio/wav" {
			finalFilePath = filepath.Join(dirPath, fmt.Sprintf("%s.mp3", fileID.String()))
			cmd := exec.Command("ffmpeg", "-i", originalFilePath, finalFilePath)
			if err := cmd.Run(); err != nil {
				log.Printf("Error converting WAV to MP3: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to convert WAV to MP3",
				})
			}
			// Удаляем оригинальный файл WAV
			if err := os.Remove(originalFilePath); err != nil {
				log.Printf("Error deleting original WAV file %s: %v", originalFilePath, err)
			}
		}

		fileRecord := models.FilePost{
			ID:     fileID,
			URL:    finalFilePath,
			PostID: post.ID,
		}
		post.Files = append(post.Files, fileRecord)
	}

	if err := initializers.DB.Create(&post).Error; err != nil {
		log.Printf("Error creating post: %v", err)
		for _, file := range post.Files {
			if err := os.Remove(file.URL); err != nil {
				log.Printf("Error deleting file %s: %v", file.URL, err)
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot create post",
		})
	}

	go utils.NotifyClientsAboutNewPost(*post)

	return c.Status(fiber.StatusCreated).JSON(post)
}

func GetUserPosts(c *fiber.Ctx) error {
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	var posts []models.Post
	query := initializers.DB.Where("user_id = ?", userResponse.ID).
		Preload("Files").
		Preload("Likes").
		Preload("Comments").
		Preload("Tags").
		Order("created_at DESC")

	tag := c.Query("tag")
	if tag != "" {
		query = query.Joins("JOIN post_tags ON post_tags.post_id = posts.id").
		Joins("JOIN tags ON tags.id = post_tags.tag_id").
		Where("tags.name = ?", tag)
	}


	// Пагинация
	err := utils.Paginate(c, query, &posts)
	if err != nil {
		return err
	}

	return nil

}

func DeletePost(c *fiber.Ctx) error {
	// Получение идентификатора поста из параметров запроса
	postID := c.Params("id")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Получение информации о пользователе из контекста
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	// Поиск поста в базе данных
	var post models.Post
	if err := initializers.DB.Where("id = ? AND user_id = ?", postID, userResponse.ID).Preload("Files").First(&post).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	if err := initializers.DB.Where("post_id = ?", postID).Delete(&models.CommentPost{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete associated comments",
		})
	}

	// Удаление лайков, связанных с постом
	if err := initializers.DB.Where("post_id = ?", postID).Delete(&models.LikePost{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete associated likes",
		})
	}

	config, _ := initializers.LoadConfig(".")
	dirPath := filepath.Join(config.IMGStorePath, userResponse.Storage)

	// Удаление файлов с диска
	for _, file := range post.Files {
		// Удаляем файл с диска
		filePath := filepath.Join(dirPath, filepath.Base(file.URL))
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Error deleting file %s: %v\n", filePath, err)
		}

		// Удаляем запись о файле из базы данных
		if err := initializers.DB.Delete(&file).Error; err != nil {
			fmt.Printf("Error deleting file record from database: %v\n", err)
		}
	}

	// Удаление связей с тегами
	if err := initializers.DB.Where("post_id = ?", postID).Delete(&models.PostTag{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete associated tags",
		})
	}	

	// Удаление поста из базы данных
	if err := initializers.DB.Delete(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete post",
		})
	}


	go utils.NotifyClientsAboutDeletedPost(post.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Post deleted successfully",
	})
}

func UpdatePost(c *fiber.Ctx) error {
	// Получение идентификатора поста из параметров запроса
	postID := c.Params("id")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Получение информации о пользователе из контекста
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	// Поиск поста в базе данных
	var post models.Post
	if err := initializers.DB.Where("id = ? AND user_id = ?", postID, userResponse.ID).Preload("Tags").First(&post).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	// Структура для получения данных из запроса
	var updateData struct {
		Content string `json:"content"`
		Tags    string `json:"tags"` // Поле для тегов
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Обновление контента поста
	if updateData.Content != "" {
		post.Content = updateData.Content
	}

	// Обновление тегов, если они переданы
	if updateData.Tags != "" {
		tagNamesArray := strings.Split(updateData.Tags, ",")
		tags := []models.Tag{}
		existingTags := []models.Tag{}
		initializers.DB.Where("name IN ?", tagNamesArray).Find(&existingTags)
		existingTagMap := make(map[string]models.Tag)
		for _, tag := range existingTags {
			existingTagMap[tag.Name] = tag
		}

		for _, tagName := range tagNamesArray {
			tagName = strings.TrimSpace(tagName)
			if len(tagName) == 0 || len(tagName) > 100 {
				continue // Пропуск пустых и слишком длинных тегов
			}
			if tag, found := existingTagMap[tagName]; found {
				tags = append(tags, tag)
			} else {
				newTag := models.Tag{
					ID:   uuid.NewV4(),
					Name: tagName,
				}
				tags = append(tags, newTag)
				if err := initializers.DB.Create(&newTag).Error; err != nil {
					log.Printf("Error creating tag %s: %v", tagName, err)
				}
			}
		}
		post.Tags = tags
	}

	// Сохранение изменений в базе данных
	if err := initializers.DB.Save(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update post",
		})
	}

	go utils.NotifyClientsAboutUpdatedPost(post)

	return c.Status(fiber.StatusOK).JSON(post)
}


func AddComment(c *fiber.Ctx) error {
	// Получение идентификатора поста из параметров запроса
	postID := c.Params("id")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Получение информации о пользователе из контекста
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	// Парсинг комментария из тела запроса
	type AddCommentRequest struct {
		Content string `json:"content"`
	}

	var commentData AddCommentRequest
	if err := c.BodyParser(&commentData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Создание нового комментария
	comment := models.CommentPost{
		ID:        uuid.NewV4(),
		PostID:    uuid.FromStringOrNil(postID),
		UserID:    userResponse.ID,
		Content:   commentData.Content,
		CreatedAt: time.Now(),
	}

	// Сохранение комментария в базе данных
	if err := initializers.DB.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add comment",
		})
	}

	// Предзагрузка данных о пользователе, чтобы вернуть полный объект комментария
	if err := initializers.DB.Preload("User").First(&comment, comment.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load user information",
		})
	}

	go utils.NotifyClientsAboutNewComment(comment)

	return c.Status(fiber.StatusCreated).JSON(comment)
}

func GetComments(c *fiber.Ctx) error {
	// Получение идентификатора поста из параметров запроса
	postID := c.Params("id")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Получение комментариев для данного поста
	var comments []models.CommentPost
	if err := initializers.DB.Where("post_id = ?", postID).
		Preload("User").Order("created_at ASC").Find(&comments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get comments",
		})
	}

	return c.Status(fiber.StatusOK).JSON(comments)
}

func DeleteComment(c *fiber.Ctx) error {
	// Получение идентификаторов поста и комментария из параметров запроса
	postID := c.Params("id") // Используйте "id" для postId
	commentID := c.Params("commentId")

	if postID == "" || commentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID and Comment ID are required",
		})
	}

	// Получение информации о пользователе из контекста
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	// Поиск комментария в базе данных
	var comment models.CommentPost
	if err := initializers.DB.Where("id = ? AND post_id = ? AND user_id = ?", commentID, postID, userResponse.ID).First(&comment).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Comment not found",
		})
	}

	// Удаление комментария из базы данных
	if err := initializers.DB.Delete(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete comment",
		})
	}

	go utils.NotifyClientsAboutDeletedComment(postID, commentID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Comment deleted successfully",
	})
}

func ToggleLike(c *fiber.Ctx) error {
	// Получение идентификатора поста из параметров запроса
	postID := c.Params("id")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Получение информации о пользователе из контекста
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	// Проверка, существует ли лайк от этого пользователя для этого поста
	var like models.LikePost
	if err := initializers.DB.Where("post_id = ? AND user_id = ?", postID, userResponse.ID).First(&like).Error; err == nil {
		// Лайк найден, удаляем его
		if err := initializers.DB.Delete(&like).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to remove like",
			})
		}

		go utils.NotifyClientsAboutLike(like, false)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Like removed successfully",
		})
	}

	// Лайк не найден, создаем новый
	like = models.LikePost{
		ID:     uuid.NewV4(),
		PostID: uuid.FromStringOrNil(postID),
		UserID: userResponse.ID,
	}

	if err := initializers.DB.Create(&like).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add like",
		})
	}

	go utils.NotifyClientsAboutLike(like, true)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Like added successfully",
	})
}

func GetPostByID(c *fiber.Ctx) error {
	// Получение идентификатора поста из параметров запроса
	postID := c.Params("id")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post ID is required",
		})
	}

	// Поиск поста в базе данных
	var post models.Post
	if err := initializers.DB.Where("id = ?", postID).
		Preload("Files").
		Preload("Likes").
		Preload("Comments").
		First(&post).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(post)
}

func GetUserAndFollowingsPosts(c *fiber.Ctx) error {
	userResponse, ok := c.Locals("user").(models.UserResponse)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot get user information",
		})
	}

	// Получение списка пользователей, на которых подписан текущий пользователь
	var user models.User
	if err := initializers.DB.Preload("Followers").Where("id = ?", userResponse.ID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot find user",
		})
	}

	// Сбор ID всех пользователей, включая самого пользователя и тех, на кого он подписан
	var userIds []uuid.UUID
	userIds = append(userIds, user.ID) // Добавление ID самого пользователя
	for _, following := range user.Followers {
		userIds = append(userIds, following.ID)
	}

	// Получение постов всех пользователей по их ID
	var posts []models.Post
	query := initializers.DB.Where("user_id IN ?", userIds).
		Preload("Files").
		Preload("Likes").
		Preload("Comments").
		Preload("User").
		Preload("Tags").
		Order("created_at DESC")

	tag := c.Query("tag")
	if tag != "" {
		// Убедитесь, что тег начинается с #
		if !strings.HasPrefix(tag, "#") {
			tag = "#" + tag
		}

		query = query.Joins("JOIN post_tags ON post_tags.post_id = posts.id").
			Joins("JOIN tags ON tags.id = post_tags.tag_id").
			Where("tags.name = ?", tag)
	}
	
	// Пагинация
	err := utils.Paginate(c, query, &posts)
	if err != nil {
		return err
	}

	return nil
}
