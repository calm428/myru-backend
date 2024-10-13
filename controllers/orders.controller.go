package controllers

import (
	"fmt"
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

	// Получаем заказы вместе с товарами для продавца
	err := utils.Paginate(c, initializers.DB.Preload("OrderItems").Where("seller_id = ?", user.ID).Order("created_at DESC"), &orders)
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

	fmt.Println(orderReq)


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

func AddDeliveryAddress(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)

	var newAddress models.DeliveryAddress
	if err := c.BodyParser(&newAddress); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Неверные данные",
		})
	}

	// Указываем ID пользователя
	newAddress.UserID = user.ID

	// Генерируем UUID для нового адреса
	newAddress.ID = uuid.NewV4()

	// Сохраняем адрес в базу данных
	if err := initializers.DB.Create(&newAddress).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Ошибка при сохранении адреса",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Адрес доставки добавлен",
		"data":    newAddress,
	})
}

func UpdateDeliveryAddress(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)
	addressID := c.Params("id") // Получаем ID адреса из параметров URL

	var address models.DeliveryAddress

	// Проверяем, существует ли адрес и принадлежит ли он текущему пользователю
	if err := initializers.DB.Where("id = ? AND user_id = ?", addressID, user.ID).First(&address).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Адрес не найден или доступ запрещен",
		})
	}

	// Обновляем адрес с новыми данными
	if err := c.BodyParser(&address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Неверные данные",
		})
	}

	// Сохраняем обновления
	if err := initializers.DB.Save(&address).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Ошибка при обновлении адреса",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Адрес успешно обновлен",
		"data":    address,
	})
}

func DeleteDeliveryAddress(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)
	addressID := c.Params("id") // Получаем ID адреса из параметров URL

	var address models.DeliveryAddress

	// Проверяем, существует ли адрес и принадлежит ли он текущему пользователю
	if err := initializers.DB.Where("id = ? AND user_id = ?", addressID, user.ID).First(&address).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Адрес не найден или доступ запрещен",
		})
	}

	// Удаляем адрес
	if err := initializers.DB.Delete(&address).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Ошибка при удалении адреса",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Адрес успешно удален",
	})
}

func GetDeliveryAddresses(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserResponse)

	var addresses []models.DeliveryAddress

	// Получаем список адресов доставки для пользователя
	if err := initializers.DB.Where("user_id = ?", user.ID).Find(&addresses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Ошибка при получении адресов",
		})
	}

	// Возвращаем список адресов, даже если он пустой
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   addresses, // Если адресов нет, возвращаем пустой массив
	})
}