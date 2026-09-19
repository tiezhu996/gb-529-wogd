package model

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"size:160;not null;uniqueIndex"`
	DisplayName  string    `json:"display_name" gorm:"size:100;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Role         string    `json:"role" gorm:"size:24;not null;check:role IN ('process_analyst','reviewer','admin')"`
	Active       bool      `json:"active" gorm:"not null;default:true"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditEvent struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	RequestID  string    `json:"request_id" gorm:"size:64;not null;index"`
	UserID     uint      `json:"user_id" gorm:"not null;index"`
	ActorEmail string    `json:"actor_email" gorm:"size:160;not null;index"`
	Action     string    `json:"action" gorm:"size:80;not null;index"`
	EntityType string    `json:"entity_type" gorm:"size:60;not null;index"`
	EntityID   uint      `json:"entity_id" gorm:"not null;index"`
	BeforeJSON string    `json:"before_json" gorm:"type:text;not null"`
	AfterJSON  string    `json:"after_json" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null;index"`
}

func (User) TableName() string       { return "users" }
func (AuditEvent) TableName() string { return "audit_events" }
