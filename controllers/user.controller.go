package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgtype"
	"gorm.io/gorm"

	"hyperpage/initializers"
	"hyperpage/models"
	"hyperpage/utils"
)

func ChangeNickName(c *fiber.Ctx) error {
	newName := strings.ReplaceAll(c.Query("new_name"), " ", "")
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	newName = reg.ReplaceAllString(newName, "")

	user := c.Locals("user").(models.UserResponse)
	userID := user.ID

	updateFields := map[string]interface{}{
		"name": newName,
	}

	// Пытаемся обновить имя пользователя
	if err := initializers.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updateFields).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Пользователь не найден"})
		}
		if strings.Contains(err.Error(), "duplicate") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Это имя уже занято"})
		}
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

// Вспомогательная функция для проверки наличия элемента в срезе
func contains(slice []string, item string) bool {
    for _, a := range slice {
        if a == item {
            return true
        }
    }
    return false
}

func ChangePhoto(c *fiber.Ctx) error {
	// Получаем пользователя из локальных данных контекста
	user := c.Locals("user").(models.UserResponse)
	userID := user.ID
	userStorage := user.Storage

	
	// Получаем загруженный файл из формы с ключом "photo"
	file, err := c.FormFile("photo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Не удалось получить файл. Убедитесь, что поле называется 'photo' и файл загружен.",
		})
	}

	// Проверяем тип файла (допустимые расширения)
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif"}
	extension := strings.ToLower(filepath.Ext(file.Filename))
	if !contains(allowedExtensions, extension) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Недопустимый тип файла. Разрешены только .jpg, .jpeg, .png, .gif",
		})
	}

	config, err := initializers.LoadConfig(".")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Ошибка загрузки конфигурации.",
		})
	}
	// Генерируем уникальное имя для файла, чтобы избежать конфликтов
	newFileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), extension)
	// Определяем путь для сохранения файла
	filePath := filepath.Join(config.IMGStorePath, user.Storage, newFileName)

	// // Убедитесь, что директория существует, если нет — создайте её
	// if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"status":  "error",
	// 		"message": "Не удалось создать директорию для хранения файла.",
	// 	})
	// }

	
    // Сохраняем файл на сервере в указанной директории
    if err := c.SaveFile(file, filePath); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "status":  "error",
            "message": "Не удалось сохранить файл. Попробуйте позже.",
        })
    }

    // Опционально: удаляем старую фотографию, если она существует и не пустая
    if user.Photo != "" {
        oldPhotoPath := filepath.Join(config.IMGStorePath, userStorage, filepath.Base(user.Photo))
        if _, err := os.Stat(oldPhotoPath); err == nil {
            if err := os.Remove(oldPhotoPath); err != nil {
                // Логируйте ошибку, но не возвращайте её пользователю
                fmt.Printf("Не удалось удалить старую фотографию: %v\n", err)
            }
        }
    }

	filePathforDB := filepath.Join(user.Storage, newFileName)

	// Обновляем поле Photo в базе данных
	updateFields := map[string]interface{}{
		"Photo": filePathforDB,
	}


    if err := initializers.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updateFields).Error; err != nil {
        // Если возникла ошибка при обновлении, удаляем только что загруженный файл, чтобы не хранить ненужные файлы
        os.Remove(filePath)

        if errors.Is(err, gorm.ErrRecordNotFound) {
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
                "status":  "error",
                "message": "Пользователь не найден",
            })
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "status":  "error",
            "message": "Произошла ошибка при обновлении фотографии",
        })
    }
	// Возвращаем успешный ответ с новым путем к фотографии
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"photo":  filePath,
	})
}

func SetTokenIOSdevice(c *fiber.Ctx) error {
	// Define a struct to parse the JSON body
	type requestBody struct {
		TokenDevice string `json:"tokenDevice"`
		// VoIP        string `json:"voipToken"`
	}

	var body requestBody

	// Parse the JSON body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid request body"})
	}

	// Validate the tokenDevice
	tokenDevice := strings.TrimSpace(body.TokenDevice)
	if tokenDevice == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "TokenDevice cannot be empty"})
	}

	// Validate the tokenDevice
	// voIPtoken := strings.TrimSpace(body.VoIP)
	// if voIPtoken == "" {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "TokenDevice VOIP cannot be empty"})
	// }

	user := c.Locals("user").(models.UserResponse)
	userID := user.ID

	updateFields := map[string]interface{}{
		"device_ios": tokenDevice,
		// "device_iosvo_ip": voIPtoken,
	}

	// Try to update the user's token device
	if err := initializers.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updateFields).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "User not found"})
		}
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

func GetMeH(id string, userName string, fileURL string, tId int64) (*models.User, error) {
	config, _ := initializers.LoadConfig(".")

	var user models.User

	err := initializers.DB.Where("telegram_token = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}

	// Open the file URL for reading
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Create a file with a unique name in the specified directory
	fileName := filepath.Base(fileURL)
	filePath := filepath.Join(config.IMGStorePath, user.Storage, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Copy the contents of the file URL to the created file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, err
	}

	user.TelegramName = &userName
	user.Tid = tId
	user.TelegramActivated = true
	user.Photo = user.Storage + `/` + fileName

	err = utils.SendPersonalMessageToClient(user.Session, "Activated")
	if err != nil {
		// handle error
		_ = err
	}

	if err := initializers.DB.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func MyTime(c *fiber.Ctx) error {
	sessionID := c.Query("session")
	var user models.User
	err := initializers.DB.First(&user, "id = ?", sessionID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"status": "fail", "message": "the user belonging to this token no longer exists"})
		} else {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"status": "fail", "message": err.Error()})
		}
	}

	// user := c.Locals("user").(models.UserResponse)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": fiber.Map{"time": user.OnlineHours}})
}

func calculateDirSize(dirPath string) (float64, error) {
	var totalSize int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	sizeInMB := float64(totalSize) / (1024 * 1024)
	return sizeInMB, nil
}

func Plan(c *fiber.Ctx) error {
	userId := c.Locals("user")
	userResp := userId.(models.UserResponse)
	userObj := models.User{
		ID: userResp.ID,
	}

	user := new(models.User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	var amount float64
	var limitstorage int

	// Add an amount variable to store the amount for the blog post
	switch user.Name {
	case "Начальный":
		amount = 150
		limitstorage = 300
	case "Бизнесс":
		amount = 500
		limitstorage = 600
	case "Расширенный":
		amount = 1000
		limitstorage = 900
	}

	// Fetch the current balance
	var billing models.Billing
	err := initializers.DB.Where("user_id = ?", userObj.ID).First(&billing).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to fetch balance",
		})
	}

	// Check if the balance is sufficient
	if billing.Amount < amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Insufficient balance",
		})
	}
	// Update the amount field in the balance table
	err = initializers.DB.Model(&models.Billing{}).
		Where("user_id = ?", userObj.ID).
		Updates(map[string]interface{}{
			"amount": gorm.Expr("amount - ?", amount),
		}).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update balance",
		})
	}

	transaction := models.Transaction{
		UserID:      userObj.ID,
		Total:       "0", // Update with the appropriate value
		Amount:      amount,
		Description: "Оплата за тариф на месяц",
		Module:      "CodeUsed",
		Type:        "deduction",
		Status:      "CLOSED_1",
	}

	err = initializers.DB.Create(&transaction).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to create transaction",
		})
	}

	// Update the user's plan, signed, and expired_plan_at columns
	err = initializers.DB.Model(&models.User{}).
		Where("id = ?", userObj.ID).
		Updates(map[string]interface{}{
			"plan":            user.Name,
			"signed":          true,
			"limit_storage":   limitstorage,
			"expired_plan_at": time.Now().AddDate(0, 0, 31),
		}).
		Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update user plan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   "GOOD",
	})
}

func AddBalance(c *fiber.Ctx) error {

	userId := c.Locals("user")
	userResp := userId.(models.UserResponse)
	userObj := models.User{
		ID: userResp.ID,
	}

	code := new(models.Codes)
	if err := c.BodyParser(code); err != nil {

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	//Try find code
	err := initializers.DB.Table("codes").Where("code = ?", code.Code).First(code).Error
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid code ID",
		})
	}

	// Check if code is already activated
	if code.Activated {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Code is already activated",
		})
	}

	// Update the amount field in the balance table
	err = initializers.DB.Model(&models.Billing{}).
		Where("user_id = ?", userObj.ID).
		Updates(map[string]interface{}{
			"amount": gorm.Expr("amount + ?", code.Balance),
		}).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update balance",
		})
	}

	// Set columns: activated, user, and used
	code.Activated = true
	code.UserId = userObj.ID
	code.Used = uint64(time.Now().Unix())

	// Update the code record in the database
	err = initializers.DB.Table("codes").Where("code = ?", code.Code).Updates(code).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update code",
		})
	}

	balanceStr := code.Balance
	balance, _ := strconv.ParseFloat(balanceStr, 64)
	transaction := models.Transaction{
		UserID:      userObj.ID,
		Total:       `0`,
		Amount:      balance,
		Description: `Пополнение баланса`,
		Module:      `CodeUsed`,
		Type:        `profit`,
		Status:      `CLOSED_1`,
	}

	initializers.DB.Create(&transaction)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   code.Balance,
	})
}

func GetMe(c *fiber.Ctx) error {
	config, _ := initializers.LoadConfig(".")
	language := c.Query("language")

	if language == "" {
		language = "en"
	}

	user := c.Locals("user").(models.UserResponse)

	billing := models.Billing{}
	result := initializers.DB.Where("user_id = ?", user.ID).First(&billing)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "failed to get balance"})
	}

	balance := billing.Amount

	dirPath := filepath.Join(config.IMGStorePath, user.Storage)
	dirSize, err := calculateDirSize(dirPath)
	if err != nil {
		fmt.Printf("Error calculating directory size: %v\n", err)
		return err
	}

	fmt.Printf("Directory size: %.2f MB\n", dirSize)

	// update session ID in the user table

	Time := models.User{}

	userTime := initializers.DB.Where("id = ?", user.ID).First(&Time)

	if userTime.Error != nil {
		// Handle the error
		fmt.Println("Error occurred:", userTime.Error)
		// Return an appropriate response or error message
	}

	roundedSize := math.Round(dirSize*10) / 10

	// var profile models.Profile
	// if err := initializers.DB.Preload("Guilds").Preload("Hashtags").Preload("City").Preload("Photos").First(&profile, "user_id = ?", user.ID).Error; err != nil {
	// 	if err == gorm.ErrRecordNotFound {
	// 		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
	// 			"status":  "error",
	// 			"message": "Profile not found",
	// 		})
	// 	}
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"status":  "error",
	// 		"message": "Failed to retrieve profile",
	// 	})
	// }
	// fmt.Println(user)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": fiber.Map{"user": user, "balance": balance, "storage": roundedSize}})

}

func extractDirectoryName(path string) string {
	// Unmarshal the path as JSON
	var pathInfo struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(path), &pathInfo); err != nil {
		// Handle the error
		return ""
	}

	// Split the path and get the first part as the directory name
	parts := strings.Split(pathInfo.Path, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

func deleteDirectory(directoryName string) error {
	config, _ := initializers.LoadConfig(".")

	// Construct the full path to the directory on your server
	// fullPath := "/path/to/your/server/" + directoryName
	fullPath := filepath.Join(config.IMGStorePath, directoryName)

	// Perform the directory deletion
	err := os.RemoveAll(fullPath)
	if err != nil {
		return err
	}

	return nil
}

// Define the route for deleting a user and its relations
func DeleteUserWithRelations(c *fiber.Ctx) error {
	userId := c.Locals("user").(models.UserResponse)

	// Fetch the user from the database
	var user models.User
	if err := initializers.DB.
		Preload("Billing").
		Preload("Profile").
		Preload("Blogs").
		First(&user, "id = ?", userId.ID).Error; err != nil {
		// Handle the error (e.g., user not found)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	// Begin a database transaction
	tx := initializers.DB.Begin()

	var profileID string
	if err := initializers.DB.Model(&models.Profile{}).Where("user_id = ?", userId.ID).Select("id").Row().Scan(&profileID); err != nil {
		// Handle the error (e.g., profile not found)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User's profile not found"})
	}

	// Fetch the paths for profile_photos to be deleted
	var files []string
	if err := initializers.DB.Table("profile_photos").Where("profile_id = ?", profileID).Pluck("files", &files).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve profile photo paths"})
	}

	// Delete the directories on the server
	for _, path := range files {
		directoryName := extractDirectoryName(path)
		if directoryName != "" {
			err := deleteDirectory(directoryName)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete directories on the server"})
			}
		}
	}
	// Delete related entities using the profileID and user.ID before deleting the user
	relatedEntities := []string{
		"profiles_guilds",
		"profiles_city",
		"profiles_hashtags",
		"profile_photos",
		"billings",
		"online_storages",
		"transactions",
		"blogs",
		"user_relation",
		"votes",
		"codes",
		"domains",
		"payments",
	}
	for _, table := range relatedEntities {
		whereColumn := "user_id"
		id := user.ID.String()
		if table == "profiles_guilds" || table == "profiles_city" || table == "profiles_hashtags" || table == "profile_photos" {
			whereColumn = "profile_id"
			id = profileID
		}

		if err := tx.Exec("DELETE FROM "+table+" WHERE "+whereColumn+" = ?", id).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete related entities from " + table})
		}

		if table == "user_relation" {
			whereColumn = "following_id"
			if err := tx.Exec("DELETE FROM "+table+" WHERE "+whereColumn+" = ?", id).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete related entities from " + table})
			}
		}
	}

	// Delete the associated profiles records
	if err := tx.Delete(&user.Profile).Error; err != nil {
		// Rollback the transaction if an error occurs
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user's profiles"})
	}

	// Delete the user and its related records
	if err := tx.Delete(&user).Error; err != nil {
		// Rollback the transaction if an error occurs
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user"})
	}

	// Commit the transaction
	tx.Commit()

	// Return a success response
	return c.JSON(fiber.Map{"message": "User and related profiles deleted successfully"})
}

// Function to delete all user accounts where IsBot is true, along with their related records
func DeleteAllBotUsersWithRelations(c *fiber.Ctx) error {
	// Find all bot users
	var botUsers []models.User
	if err := initializers.DB.Where("is_bot = ?", true).Find(&botUsers).Error; err != nil {
		// Handle the error (e.g., failed to fetch bot users)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch bot users"})
	}

	// Begin a database transaction
	tx := initializers.DB.Begin()

	// Loop through each bot user and delete
	for _, user := range botUsers {
		var profileID string
		if err := tx.Model(&models.Profile{}).Where("user_id = ?", user.ID).Select("id").Scan(&profileID).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find profile"})
		}

		// Fetch the paths for profile_photos to be deleted
		var files []string
		if err := tx.Table("profile_photos").Where("profile_id = ?", profileID).Pluck("path", &files).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile photos"})
		}

		// Delete the directories on the server for profile_photos
		for _, path := range files {
			directoryName := extractDirectoryName(path)
			if directoryName != "" {
				if err := deleteDirectory(directoryName); err != nil {
					tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete directory"})
				}
			}
		}

		// Delete related entities using the profileID and user.ID before deleting the user
		relatedEntities := []string{
			"profiles_guilds",
			"profiles_city",
			"profiles_hashtags",
			"profile_photos",
			"billings",
			"online_storages",
			"transactions",
			"blogs",
			"user_relation",
			"votes",
			"codes",
			"domains",
			"payments",
		}
		for _, table := range relatedEntities {
			whereColumn := "user_id"
			id := user.ID.String()
			if table == "profiles_guilds" || table == "profiles_city" || table == "profiles_hashtags" || table == "profile_photos" {
				whereColumn = "profile_id"
				id = profileID
			}

			if err := tx.Exec("DELETE FROM "+table+" WHERE "+whereColumn+" = ?", id).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete related entities from " + table})
			}

			if table == "user_relation" {
				whereColumn = "following_id"
				if err := tx.Exec("DELETE FROM "+table+" WHERE "+whereColumn+" = ?", id).Error; err != nil {
					tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete related entities from " + table})
				}
			}
		}

		// Delete the associated profiles records
		if err := tx.Delete(&user.Profile).Error; err != nil {
			// Rollback the transaction if an error occurs
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user's profiles"})
		}

		// Delete the user itself
		if err := tx.Delete(&user).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user"})
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit deletion of bot users"})
	}

	// Return a success response
	return c.JSON(fiber.Map{"message": "All bot users deleted successfully"})
}

func GetMeFirst(c *fiber.Ctx) error {

	user := c.Locals("user").(models.UserResponse)

	billing := models.Billing{}
	result := initializers.DB.Where("user_id = ?", user.ID).First(&billing)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "failed to get balance"})
	}

	balance := billing.Amount

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": fiber.Map{"user": user, "balance": balance}})

}

func SetVipUser(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)
	amount := c.Get("Amount")

	priceFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid price value",
		})
	}

	// Fetch the current balance
	var billing models.Billing
	err = initializers.DB.Where("user_id = ?", user.ID).First(&billing).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to fetch balance",
		})
	}

	// Check if the balance is sufficient
	if billing.Amount < priceFloat {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Insufficient balance",
		})
	}
	// Update the amount field in the balance table
	err = initializers.DB.Model(&models.Billing{}).
		Where("user_id = ?", user.ID).
		Updates(map[string]interface{}{
			"amount": gorm.Expr("amount - ?", priceFloat),
		}).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update balance",
		})
	}

	transaction := models.Transaction{
		UserID:      user.ID,
		Total:       "0", // Update with the appropriate value
		Amount:      priceFloat,
		Description: "Оплата за активацию сайта",
		Module:      "site",
		Type:        "deduction",
		Status:      "CLOSED_1",
	}

	err = initializers.DB.Create(&transaction).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to create transaction",
		})
	}

	// Update the user's role to "VIP"
	result := initializers.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("role", "vip")
	if result.Error != nil {
		// Handle the error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user role",
		})
	}

	if result.RowsAffected == 0 {
		// User not found
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Create domain settings
	settings := pgtype.JSONB{}
	settingsMap := map[string]interface{}{
		"logo":      "logo.png",
		"title":     "Мой веб-сайт",
		"metadescr": "Добро пожаловать на мой сайт!",
		// ... other settings ...
	}
	if err := settings.Set(settingsMap); err != nil {
		// Handle error if needed
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create domain settings",
		})
	}

	if priceFloat == 2500 {
		newExpiredAt := time.Now().AddDate(0, 0, 30)

		// Create a new domain record
		domain := models.Domain{
			UserID:    user.ID,
			Username:  user.Name,
			Name:      strings.ToLower(user.Name) + ".myru.online",
			Settings:  settings,
			ExpiredAt: &newExpiredAt, // Assign a pointer to the newExpiredAt value
			Status:    "activated",
		}

		if err := initializers.DB.Create(&domain).Error; err != nil {
			fmt.Println("Error creating domain:", err)
			// Handle the error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create domain",
			})
		}

		// Use newExpiredAt in further processing as needed
	} else if priceFloat == 20000 {
		newExpiredAt := time.Now().AddDate(1, 0, 0)

		// Create a new domain record
		domain := models.Domain{
			UserID:    user.ID,
			Username:  user.Name,
			Name:      strings.ToLower(user.Name) + ".myru.online",
			Settings:  settings,
			ExpiredAt: &newExpiredAt, // Assign a pointer to the newExpiredAt value
			Status:    "activated",
		}

		if err := initializers.DB.Create(&domain).Error; err != nil {
			fmt.Println("Error creating domain:", err)
			// Handle the error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create domain",
			})
		}
		// Use newExpiredAt in further processing as needed
	}

	// Respond with a success message
	return c.JSON(fiber.Map{
		"message": "User role updated to VIP and domain created",
	})
}
