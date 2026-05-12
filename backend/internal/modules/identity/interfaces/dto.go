// Package interfaces hosts the Identity module's HTTP delivery layer:
// DTOs, handlers, and authentication middleware. It bridges the public
// API surface to the application service.
package interfaces

import (
	"time"

	"github.com/nebulacloud/nebula/internal/modules/identity/application"
	"github.com/nebulacloud/nebula/internal/modules/identity/domain"
)

// registerRequest is the JSON body for POST /auth/register.
type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
}

// loginRequest is the JSON body for POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// refreshRequest is the JSON body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// logoutRequest is the JSON body for POST /auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// userDTO is the public projection of domain.User.
type userDTO struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"display_name,omitempty"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	IsAdmin       bool       `json:"is_admin"`
	EmailVerified bool       `json:"email_verified"`
	MFAEnabled    bool       `json:"mfa_enabled"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func toUserDTO(u domain.User) userDTO {
	return userDTO{
		ID:            u.ID.String(),
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		AvatarURL:     u.AvatarURL,
		IsAdmin:       u.IsAdmin,
		EmailVerified: u.EmailVerified,
		MFAEnabled:    u.MFAEnabled,
		LastLoginAt:   u.LastLoginAt,
		CreatedAt:     u.CreatedAt,
	}
}

// tokenPairDTO is the response payload on login / refresh.
type tokenPairDTO struct {
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	AccessExpiry   time.Time `json:"access_expiry"`
	RefreshExpiry  time.Time `json:"refresh_expiry"`
	TokenType      string    `json:"token_type"`
	User           userDTO   `json:"user"`
}

func toTokenPairDTO(p application.TokenPair) tokenPairDTO {
	return tokenPairDTO{
		AccessToken:   p.AccessToken,
		RefreshToken:  p.RefreshToken,
		AccessExpiry:  p.AccessExpiry,
		RefreshExpiry: p.RefreshExpiry,
		TokenType:     "Bearer",
		User:          toUserDTO(p.User),
	}
}
