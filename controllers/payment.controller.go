package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"hyperpage/initializers"
	"hyperpage/models"
	"hyperpage/utils"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/nikita-vanyasin/tinkoff"
	"gorm.io/gorm"
)

func handlePanic(c *fiber.Ctx) {
	if r := recover(); r != nil {
		fmt.Println("Recovered from panic:", r)
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "An unexpected error occurred",
		})
	}
}

func Pending(c *fiber.Ctx) error {

	defer handlePanic(c)

	var requestBody map[string]interface{}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	var response = requestBody

	// Access the value of the PaymentId field
	// paymentIDFloat := response["PaymentId"].(float64)

	// Convert the PaymentId to an integer
	var paymentID int64

	switch v := requestBody["PaymentId"].(type) {
	case float64:
		paymentID = int64(v)
	case string:
		// Преобразование строки в int64
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid PaymentId format",
			})
		}
		paymentID = id
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "PaymentId is of an unexpected type",
		})
	}

	if response["Status"] == "CONFIRMED" { // Use == for comparison
		var payment models.Payments
		if err := initializers.DB.Where("payment_id = ?  AND status = ?", fmt.Sprintf("%d", paymentID), "NEW").First(&payment).Error; err != nil {
			// Handle error if the record is not found or other issues
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Payment not found or status is not 'NEW'",
			})
		}
		// Update the status to "applied"
		payment.Status = "applied"
		if err := initializers.DB.Save(&payment).Error; err != nil {
			// Handle error if the update fails
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to update payment status",
			})
		}

		decimalAmount := float64(payment.Amount) / 100.0

		// Update the amount field in the balance table
		updateBalanceErr := initializers.DB.Model(&models.Billing{}).
			Where("user_id = ?", payment.UserID).
			Updates(map[string]interface{}{
				"amount": gorm.Expr("amount + ?", decimalAmount),
			}).Error
		if updateBalanceErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to update balance",
			})
		}

		var user models.User
		if err := initializers.DB.Model(&models.User{}).Where("id = ?", payment.UserID).First(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to find user",
			})
		}

		userSession := user.Session

		transaction := models.Transaction{
			UserID:      payment.UserID,
			Total:       `0`,
			Amount:      decimalAmount,
			Description: `Пополнение баланса c карты банка`,
			Module:      `Payment`,
			Type:        `profit`,
			Status:      `CLOSED_1`,
		}

		createTransactionErr := initializers.DB.Create(&transaction).Error
		if createTransactionErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to create transaction",
			})
		}
		

		

		var err = utils.SendPersonalMessageToClient(userSession, "BalanceAdded")
		if err != nil {
			// handle error
			_ = err
		}

		err = godotenv.Load("app.env")
		if err != nil {
			log.Fatalf("Error loading .env file")
		}

		// Формирование URL с учетом userID
		blockchainAPI := os.Getenv("BLOCKCHAIN_API")
		api := fmt.Sprintf("%s/api/add_block", blockchainAPI)
		// payment.UserID

		data := url.Values{}
		data.Set("transaction_type", "fiat_conversion")
		data.Set("from_currency", "₽")
		data.Set("to_currency", "RUDT")
		data.Set("amount", fmt.Sprintf("%.2f", decimalAmount)) // Преобразуем float64 в строку
		data.Set("conversion_rate", "1")
		data.Set("data", "Конвертация Рубля в криптовалюту RUDT")
		data.Set("user_id", payment.UserID.String()) // Преобразуем UUID в строку
	

		// Получение токена для авторизации
		blockchainToken := os.Getenv("BLOCKCHAIN_TOKEN")

		if blockchainToken == "" {
			return errors.New("blockchain token is missing in environment variables")
		}
		

		// Формирование запроса
		req, err := http.NewRequest("POST", api, bytes.NewBufferString(data.Encode()))
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+blockchainToken)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Выполнение запроса
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Обработка ответа
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("ошибка при отправке запроса, статус код: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	
		fmt.Println("Response body:", string(body))
	

		// var balanceResp BalanceResponse
		// if err := json.Unmarshal(body, &balanceResp); err != nil {
		// 	return err
		// }


		}

	// return the city names as a JSON response
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   "initRes",
	})
}

func CreateInvoice(c *fiber.Ctx) error {

	// var terminalKey = "1718186727633DEMO"
	// var terminalPassword = "lkSI$mq4FqxMBrwf"

	var terminalKey = "1669511559870"
	var terminalPassword = "vfo6kjw9gntp2sxz"

	client := tinkoff.NewClient(terminalKey, terminalPassword)

	orderID := strconv.FormatInt(time.Now().UnixNano(), 10)

	user := c.Locals("user")
	sum := c.Get("amount")

	amount, err := strconv.ParseUint(sum, 10, 64)
	if err != nil {
		// Handle the error if the conversion fails
		return err
	}

	userResp := user.(models.UserResponse)

	initReq := &tinkoff.InitRequest{
		Amount:      amount,
		OrderID:     orderID,
		CustomerKey: userResp.Name,
		Description: "Пополнение баланса в профиле " + userResp.Name + " на платформе моя Россия онлайн",
		// PayType:         tinkoff.PayTypeOneStep,
		RedirectDueDate: tinkoff.Time(time.Now().Add(4 * time.Hour * 24)), // ссылка истечет через 4 дня
		Receipt: &tinkoff.Receipt{
			Email: userResp.Email,
			Items: []*tinkoff.ReceiptItem{
				{
					Price:         amount,
					Quantity:      "1",
					Amount:        amount,
					Name:          "Баланс на сумму " + strconv.FormatUint(amount, 10),
					Tax:           tinkoff.VATNone,
					PaymentMethod: tinkoff.PaymentMethodFullPayment,
					PaymentObject: tinkoff.PaymentObjectIntellectualActivity,
				},
			},
			Taxation: tinkoff.TaxationUSNIncome,
			Payments: &tinkoff.ReceiptPayments{
				Electronic: amount,
			},
		},

		//custom fields for tinkoff
		Data: map[string]string{},
	}

	initRes, err := client.Init(initReq)
	if err != nil {
		// Handle the error here, if needed
		fmt.Println("Error:", err)
	} else {

		payments := models.Payments{
			UserID:    userResp.ID,
			Amount:    float64(amount),
			Status:    "NEW",
			PaymentId: initRes.PaymentID, // Store as a string directly
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create the database record
		if err := initializers.DB.Create(&payments).Error; err != nil {
			log.Println("Could not create payment:", err)
		} else {
			fmt.Println("Payment record created successfully")
		}
	}

	// return the city names as a JSON response
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   initRes,
	})
}
