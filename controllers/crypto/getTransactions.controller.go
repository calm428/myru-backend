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

// TransactionType структура для разных типов транзакций
type TransactionType map[string]interface{}


// OnlineTime структура для типа транзакции "OnlineTime"
type OnlineTime struct {
    OnlineTime    int     `json:"online_time"`
    RewardAmount  float64 `json:"reward_amount"`
}

// TransactionResponse структура ответа для одной транзакции
type TransactionResponse struct {
    ID              string          `json:"id"`
    Hash            string          `json:"hash"`
    Data            string          `json:"data"`
    Timestamp       string          `json:"timestamp"`
    TransactionType TransactionType `json:"transaction_type"`
    UserID          string          `json:"user_id"`
    Signature       []int           `json:"signature"`
}

// TransactionsResponse структура ответа для списка транзакций
type TransactionsResponse []TransactionResponse

// Function to call blockchain API to fetch user transactions
func fetchUserTransactions(userID string) (*TransactionsResponse, error) {
    // Загрузка переменных окружения из файла .env
    err := godotenv.Load("app.env")
    if err != nil {
        log.Fatalf("Error loading .env file")
    }

    // Формирование URL с учетом userID
    blockchainAPI := os.Getenv("BLOCKCHAIN_API")
    if blockchainAPI == "" {
        return nil, errors.New("BLOCKCHAIN_API is not set in environment variables")
    }

    url := fmt.Sprintf("%s/user/%s/transactions", blockchainAPI, userID)

    // Формирование запроса
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Если токен не требуется, этот шаг можно пропустить
    // Но убедитесь, что API действительно не требует авторизации
    // req.Header.Set("Authorization", "Bearer "+blockchainToken)

    // Выполнение запроса
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Обработка ответа
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to fetch transactions: status code %d", resp.StatusCode)
    }

    // Чтение тела ответа
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    fmt.Println("Response body:", string(body))

    var transactionsResp TransactionsResponse
    if err := json.Unmarshal(body, &transactionsResp); err != nil {
        return nil, err
    }

    return &transactionsResp, nil
}
// Контроллер для получения транзакций пользователя
func GetTransactions(c *fiber.Ctx) error {
    // Извлечение информации о пользователе из контекста Fiber
    userInterface := c.Locals("user")
    user, ok := userInterface.(models.UserResponse)
    if !ok {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "status":  "error",
            "message": "user information is missing or invalid",
        })
    }

    userID := user.ID.String()

    // Запрос к Blockchain API для получения транзакций
    transactionsResp, err := fetchUserTransactions(userID)
    if err != nil {
        log.Printf("Error fetching transactions for user %s: %v", userID, err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "status":  "error",
            "message": "failed to fetch transactions",
        })
    }

    // Возвращение результата
    return c.JSON(fiber.Map{
        "status":  "success",
        "data":    transactionsResp,
    })
}



