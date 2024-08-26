package utils

import (
	"fmt"
	"hyperpage/initializers"
	"hyperpage/models"
	"log"
	"regexp"
	"strings"

	uuid "github.com/satori/go.uuid"
)

func main() {

	// Получение всех постов
	var posts []models.Post
	if err := initializers.DB.Preload("Tags").Find(&posts).Error; err != nil {
		log.Fatalf("Ошибка при получении постов: %v", err)
	}

	for _, post := range posts {
		// Извлечение возможных тегов из контента поста
		content := post.Content
		tags := extractHashtags(content)

		if len(tags) > 0 {
			// Создание/получение тегов и связывание их с постом
			for _, tagName := range tags {
				var tag models.Tag
				// Поиск существующего тега
				if err := initializers.DB.Where("name = ?", tagName).First(&tag).Error; err != nil {
					// Если тег не найден, создаем новый
					tag = models.Tag{
						ID:   uuid.NewV4(),
						Name: tagName,
					}
					if err := initializers.DB.Create(&tag).Error; err != nil {
						log.Printf("Ошибка при создании тега %s: %v", tagName, err)
						continue
					}
				}

				// Проверка, связан ли уже тег с постом
				alreadyLinked := false
				for _, existingTag := range post.Tags {
					if existingTag.ID == tag.ID {
						alreadyLinked = true
						break
					}
				}

				// Если тег еще не связан с постом, связываем его
				if !alreadyLinked {
					if err := initializers.DB.Model(&post).Association("Tags").Append(&tag); err != nil {
						log.Printf("Ошибка при линковке поста с тегом %s: %v", tagName, err)
					}
				}
			}
		}
	}

	fmt.Println("Линковка завершена.")
}

// extractHashtags извлекает все строки, начинающиеся с #, включая кириллические и латинские символы
func extractHashtags(content string) []string {
	// Регулярное выражение для поиска хэштегов
	re := regexp.MustCompile(`(?i)#\w[\w-]*`)
	matches := re.FindAllString(content, -1)

	// Удаляем дублирующиеся хэштеги
	tagMap := make(map[string]bool)
	var tags []string
	for _, tag := range matches {
		cleanTag := strings.Trim(tag, "#.,!? ")
		if _, exists := tagMap[cleanTag]; !exists && len(cleanTag) > 0 {
			tags = append(tags, "#"+cleanTag)
			tagMap[cleanTag] = true
		}
	}

	return tags
}
