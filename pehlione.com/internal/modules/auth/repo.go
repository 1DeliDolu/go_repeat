package auth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func normEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func (r *Repo) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	u := User{
		ID:           uuid.NewString(),
		Email:        normEmail(email),
		PasswordHash: passwordHash,
		Role:         "user",
	}
	if err := r.db.WithContext(ctx).Create(&u).Error; err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *Repo) CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) (Session, error) {
	s := Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&s).Error; err != nil {
		return Session{}, err
	}
	return s, nil
}

// GetByEmail finds a user by email address.
func (r *Repo) GetByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", normEmail(email)).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create inserts a new user into the database.
func (r *Repo) Create(user *User) error {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	user.Email = normEmail(user.Email)
	if user.Role == "" {
		user.Role = "user"
	}
	return r.db.Create(user).Error
}

// GetByID finds a user by ID.
func (r *Repo) GetByID(ctx context.Context, id string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdatePassword updates a user's password hash.
func (r *Repo) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}

// UpdateUser updates a user's profile information.
func (r *Repo) UpdateUser(ctx context.Context, user *User) error {
	if user.Email != "" {
		user.Email = normEmail(user.Email)
	}
	return r.db.WithContext(ctx).Model(user).Updates(user).Error
}
