package crypto

import (
	"encoding/json"
	"errors"
	"fmt"
	"hyperpage/models"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// Blockchain API response structure
type WalletResponse struct {
    Address string `json:"address"`
	Public_key string `json:"public_key"`
}

// Function to call blockchain API
func fetchUserWallet(userID string) (*WalletResponse, error) {
	err := godotenv.Load("app.env")
    if err != nil {
        log.Fatalf("Error loading .env file")
    }

	// Формирование URL с учетом userID
	blockchainAPI := os.Getenv("BLOCKCHAIN_API")
	url := fmt.Sprintf("%s/api/user/%s/generate_wallet", blockchainAPI, userID)

	// Получение токена для авторизации
	blockchainToken := os.Getenv("BLOCKCHAIN_TOKEN")

	if blockchainToken == "" {
		return nil, errors.New("blockchain token is missing in environment variables")
	}
	

	// Формирование запроса
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+blockchainToken)

	// Выполнение запроса
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Обработка ответа
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch wallet from blockchain API")
	}

	// Используем io.ReadAll вместо ioutil.ReadAll
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println("Response body:", string(body))


	var WalletResp WalletResponse
	if err := json.Unmarshal(body, &WalletResp); err != nil {
		return nil, err
	}

	return &WalletResp, nil
}

// Основная функция для получения баланса
func Wallet(c *fiber.Ctx) error {
	
	// Извлечение user_id из контекста Fiber

	userInterface := c.Locals("user")
	user, ok := userInterface.(models.UserResponse)
	if !ok {
		return errors.New("user information is missing or invalid")
	}

	userID := user.ID.String()
	
	// Запрос к blockchain API для получения баланса
	WalletResp, err := fetchUserWallet(userID)
	
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch wallet",
		})
	}

	// Возвращение результата
	return c.JSON(fiber.Map{
		"status":  "success",
		"balance": WalletResp,
	})
}
