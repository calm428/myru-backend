package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Модель для заказов
type Order struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	UserID      uuid.UUID  `gorm:"type:uuid;null;index"`  // Заказчик
	SellerID    uuid.UUID  `gorm:"type:uuid;not null;index"`  // Продавец
	TotalAmount float64    `gorm:"not null"`                  // Общая сумма заказа
	Status      string     `gorm:"type:varchar(50);default:'pending'"` // Статус заказа (pending, completed, canceled)
	CreatedAt   time.Time  `gorm:"autoCreateTime"`            // Время создания
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`            // Время обновления
	OrderItems  []OrderItem `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // Товары в заказе
	DeliveryAddressID uuid.UUID `gorm:"type:uuid;not null"`        // Ссылка на адрес доставки
	DeliveryAddress  DeliveryAddress `gorm:"foreignKey:DeliveryAddressID"`    // Связь с моделью адреса доставки

}

// Модель для товаров в заказе
type OrderItem struct {
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	OrderID  uuid.UUID `gorm:"type:uuid;not null;index"`       // Ссылка на заказ
	Product  string    `gorm:"type:varchar(255);not null"`     // Название товара
	Price    float64   `gorm:"not null"`                      // Цена товара
	Quantity int       `gorm:"not null"`                      // Количество товара
	CreatedAt time.Time `gorm:"autoCreateTime"`                // Время добавления товара
}
