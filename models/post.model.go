package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Post struct {
	ID        uuid.UUID     `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Content   string        `gorm:"type:text" json:"content"`
	UserID    uuid.UUID     `json:"user_id"`
	User      User          `gorm:"foreignKey:UserID" json:"user"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Likes     []LikePost    `json:"likes"`
	Comments  []CommentPost `json:"comments"`
	Files     []FilePost    `json:"files"`
	Tags      []Tag         `gorm:"many2many:post_tags;" json:"tags"`

}

type LikePost struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	PostID uuid.UUID `json:"post_id"`
	UserID uuid.UUID `json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"user"`
}

type CommentPost struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Content   string    `json:"content"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type FilePost struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	URL    string    `json:"url"`
	PostID uuid.UUID `json:"post_id"`
}

type Tag struct {
	ID   uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Name string    `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Posts []Post   `gorm:"many2many:post_tags;" json:"posts"`
}

type PostTag struct {
    PostID uuid.UUID `gorm:"type:uuid;primaryKey" json:"post_id"`
    TagID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
}