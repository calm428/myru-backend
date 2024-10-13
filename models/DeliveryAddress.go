package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type DeliveryAddress struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index"`  // Связь с пользователем
	AddressName string     `gorm:"type:varchar(255);not null"`  // Название адреса (например, "Дом" или "Офис")
	City        string     `gorm:"type:varchar(100);not null"`  // Город
	Street      string     `gorm:"type:varchar(255);not null"`  // Улица
	Building    string     `gorm:"type:varchar(50);not null"`   // Номер дома
	Apartment   string     `gorm:"type:varchar(50);null"`       // Номер квартиры/офиса (если есть)
	Entrance    string     `gorm:"type:varchar(50);null"`       // Подъезд (если есть)
	Floor       string     `gorm:"type:varchar(50);null"`       // Этаж (если есть)
	Intercom    string     `gorm:"type:varchar(50);null"`       // Код домофона (если есть)
	PostalCode  string     `gorm:"type:varchar(20);not null"`   // Почтовый индекс
	PhoneNumber string     `gorm:"type:varchar(20);not null"`   // Контактный телефон
	CreatedAt   time.Time  `gorm:"autoCreateTime"`              // Время создания
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`              // Время обновления
}
