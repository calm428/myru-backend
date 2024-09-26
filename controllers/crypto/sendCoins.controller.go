package crypto

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// Структура для получения данных из JSON запроса
type SendCoinsRequest struct {
	FromWallet string `json:"from_wallet"`
	ToWallet   string `json:"to_wallet"`
	PublicKey  string `json:"public_key"`
	Amount     string `json:"amount"`
}

// Структура для ответа от API
type SendCoinsResponse struct {
	FromWallet      string  `json:"from_wallet"`
	ToWallet        string  `json:"to_wallet"`
	Message         string  `json:"message"`
	NewFromBalance  float64 `json:"new_from_balance"`
	NewToBalance    float64 `json:"new_to_balance"`
	Signature       string  `json:"signature"`
}

// Функция для отправки монет на blockchain API
func sendCoinsToAPI(fromWallet, toWallet, publicKey, amount string) (*SendCoinsResponse, error) {
	err := godotenv.Load("app.env")
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	// Формирование URL для отправки монет
	blockchainAPI := os.Getenv("BLOCKCHAIN_API")
	apiURL := fmt.Sprintf("%s/api/send_coins", blockchainAPI)

	// Получение токена для авторизации
	blockchainToken := os.Getenv("BLOCKCHAIN_TOKEN")
	if blockchainToken == "" {
		return nil, errors.New("blockchain token is missing in environment variables")
	}

	// Формирование данных для запроса в формате x-www-form-urlencoded
	data := url.Values{}
	data.Set("from_wallet", fromWallet)
	data.Set("to_wallet", toWallet)
	data.Set("public_key", publicKey)
	data.Set("amount", amount)

	// Формирование запроса
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+blockchainToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.PostForm = data

	// Отправка запроса
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Обработка ответа от API
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to send coins to blockchain API")
	}

	// Чтение и разбор ответа
	var sendCoinsResp SendCoinsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendCoinsResp); err != nil {
		return nil, err
	}

	return &sendCoinsResp, nil
}

// Контроллер для отправки монет
func SendCoins(c *fiber.Ctx) error {
	// Чтение данных в формате JSON из тела запроса
	var reqData SendCoinsRequest
	if err := c.BodyParser(&reqData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request format",
		})
	}

	// Проверка обязательных параметров
	if reqData.FromWallet == "" || reqData.ToWallet == "" || reqData.PublicKey == "" || reqData.Amount == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing required fields",
		})
	}

	// Отправка данных на blockchain API
	sendCoinsResp, err := sendCoinsToAPI(reqData.FromWallet, reqData.ToWallet, reqData.PublicKey, reqData.Amount)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send coins",
		})
	}

	// Возвращение успешного ответа с данными
	return c.JSON(fiber.Map{
		"status":           "success",
		"from_wallet":      sendCoinsResp.FromWallet,
		"to_wallet":        sendCoinsResp.ToWallet,
		"message":          sendCoinsResp.Message,
		"new_from_balance": sendCoinsResp.NewFromBalance,
		"new_to_balance":   sendCoinsResp.NewToBalance,
		"signature":        sendCoinsResp.Signature,
	})
}
