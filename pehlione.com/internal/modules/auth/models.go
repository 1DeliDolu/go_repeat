package auth

import "time"

type User struct {
	ID              string     `gorm:"type:char(36);primaryKey"`
	Email           string     `gorm:"type:varchar(255);not null;uniqueIndex:ux_users_email"`
	PasswordHash    string     `gorm:"type:varchar(255);not null"`
	Role            string     `gorm:"type:varchar(32);not null;default:user"`
	Status          string     `gorm:"type:varchar(16);not null;default:pending"`
	EmailVerifiedAt *time.Time `gorm:"type:datetime(3)"`
	PhoneE164       *string    `gorm:"type:varchar(32)"`
	PhoneVerifiedAt *time.Time `gorm:"type:datetime"`
	SMSOptIn        bool       `gorm:"type:tinyint(1);not null;default:0"`
	SMSOptOutAt     *time.Time `gorm:"type:datetime"`
	FirstName       *string    `gorm:"type:varchar(255)"`
	LastName        *string    `gorm:"type:varchar(255)"`
	Address         *string    `gorm:"type:text"`
	CreatedAt       time.Time  `gorm:"type:datetime(3);not null"`
	UpdatedAt       time.Time  `gorm:"type:datetime(3);not null"`
}

func (User) TableName() string { return "users" }

type Session struct {
	ID         string    `gorm:"type:char(36);primaryKey"`
	UserID     string    `gorm:"type:char(36);not null;index:ix_sessions_user_id"`
	TokenHash  []byte    `gorm:"type:binary(32);not null;uniqueIndex:ux_sessions_token_hash"`
	ExpiresAt  time.Time `gorm:"type:datetime(3);not null"`
	CreatedAt  time.Time `gorm:"type:datetime(3);not null"`
	LastSeenAt time.Time `gorm:"type:datetime(3);not null"`
}

func (Session) TableName() string { return "sessions" }
