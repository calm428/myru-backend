package controllers

import (
	"hyperpage/initializers"
	"hyperpage/models"
	"hyperpage/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/satori/go.uuid"
)

func GetOrdersForSeller(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)

	// Проверяем, является ли пользователь продавцом
	if !user.Seller {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Вы не являетесь продавцом",
		})
	}

	var orders []models.Order

	// Получаем заказы для продавца
	err := utils.Paginate(c, initializers.DB.Where("seller_id = ?", user.ID).Order("created_at DESC"), &orders)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Заказы не найдены",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   orders,
	})
}

func CreateOrder(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)

	// Структура для получения данных из запроса
	type OrderItemRequest struct {
		ID       string  `json:"id"`
		Title    string  `json:"title"`
		Price    float64 `json:"price"`
		Quantity int     `json:"quantity"`
		Image    string  `json:"image"`
		Seller   string  `json:"seller"`  // Продавец передается как строка (имя пользователя)
	}

	type CustomerDetails struct {
		Name    string `json:"name" validate:"required"`
		Email   string `json:"email" validate:"required"`
		Address string `json:"address" validate:"required"`
		Phone   string `json:"phone" validate:"required"`
	}

	type OrderRequest struct {
		CartItems      []OrderItemRequest `json:"cartItems" validate:"required"`
		CustomerDetails CustomerDetails   `json:"customerDetails" validate:"required"`
	}

	var orderReq OrderRequest
	if err := c.BodyParser(&orderReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Ошибка обработки данных",
		})
	}

	// Получаем ID продавца по его имени (seller)
	var seller models.User
	for _, item := range orderReq.CartItems {
		err := initializers.DB.Where("name = ?", item.Seller).First(&seller).Error
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Продавец не найден: " + item.Seller,
			})
		}

		// Создаем новый заказ для текущего продавца
		order := models.Order{
			ID:          uuid.NewV4(),
			UserID:      user.ID,
			SellerID:    seller.ID, // Используем ID продавца из базы данных
			TotalAmount: item.Price * float64(item.Quantity),
			Status:      "pending",
		}

		// Сохраняем заказ в базе данных
		if err := initializers.DB.Create(&order).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Не удалось создать заказ",
			})
		}

		// Добавляем товары к заказу
		orderItem := models.OrderItem{
			ID:       uuid.NewV4(),
			OrderID:  order.ID,
			Product:  item.Title,
			Price:    item.Price,
			Quantity: item.Quantity,
		}

		// Сохраняем товар в базе данных
		if err := initializers.DB.Create(&orderItem).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Ошибка при добавлении товаров в заказ",
			})
		}

		SendNotificationToOwner(
			seller.ID.String(), 
			"У вас новая продажа", 
			"На сумму " + strconv.FormatFloat(item.Price * float64(item.Quantity), 'f', 2, 64) + " руб.", 
			"https://www.myru.online/profile/posts?tabs=sales",
		)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Заказы успешно созданы",
	})
}

