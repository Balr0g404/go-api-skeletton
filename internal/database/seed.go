package database

import (
	"log"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	if count > 0 {
		return
	}

	admin := &models.User{
		Email:     "admin@admin.com",
		FirstName: "Admin",
		LastName:  "Admin",
		Role:      models.RoleAdmin,
		Active:    true,
	}
	admin.SetPassword("admin123")

	if err := db.Create(admin).Error; err != nil {
		log.Printf("failed to seed admin user: %v", err)
		return
	}

	log.Println("seeded default admin user (admin@admin.com / admin123)")
}
