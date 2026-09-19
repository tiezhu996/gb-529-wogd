package service

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
)

func TestParseTokenUsesCurrentActiveUserAndRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:auth-current-user-529?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password-529"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := model.User{Email: "analyst@example.test", DisplayName: "Analyst", PasswordHash: string(hash), Role: "process_analyst", Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	auth := NewAuthService(repository.NewSupportRepository(db), "test-secret")
	login, err := auth.Login(context.Background(), user.Email, "password-529")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{"role": "reviewer", "active": true}).Error; err != nil {
		t.Fatalf("change role: %v", err)
	}
	claims, err := auth.ParseToken(context.Background(), login.Token)
	if err != nil {
		t.Fatalf("parse token after role change: %v", err)
	}
	if claims.Role != "reviewer" {
		t.Fatalf("role = %q, want current reviewer role", claims.Role)
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("active", false).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, err := auth.ParseToken(context.Background(), login.Token); err == nil {
		t.Fatal("disabled user token remained valid")
	}
}
