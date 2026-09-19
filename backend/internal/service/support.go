package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	repo      *repository.SupportRepository
	jwtSecret []byte
}

type LoginResult struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func NewAuthService(repo *repository.SupportRepository, secret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: []byte(secret)}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	user, err := s.repo.FindUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return LoginResult{}, err
	}
	if !user.Active || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, api.NewError(401, "INVALID_CREDENTIALS", "邮箱或密码不正确")
	}
	now := time.Now().UTC()
	claims := Claims{
		UserID: user.ID, Email: user.Email, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: fmt.Sprintf("%d", user.ID), IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(8 * time.Hour)), Issuer: "lng-boiloff-gas-balance",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return LoginResult{}, fmt.Errorf("sign JWT: %w", err)
	}
	return LoginResult{Token: token, User: user}, nil
}

func (s *AuthService) ParseToken(ctx context.Context, raw string) (Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer("lng-boiloff-gas-balance"))
	if err != nil || !token.Valid {
		return Claims{}, api.NewError(401, "INVALID_TOKEN", "登录凭证无效或已过期")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return Claims{}, api.NewError(401, "INVALID_TOKEN", "登录凭证格式无效")
	}
	user, err := s.repo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return Claims{}, err
	}
	if !user.Active {
		return Claims{}, api.NewError(401, "USER_DISABLED", "用户已停用")
	}
	claims.Email = user.Email
	claims.Role = user.Role
	return *claims, nil
}

func (s *AuthService) Me(ctx context.Context, userID uint) (model.User, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return model.User{}, err
	}
	if !user.Active {
		return model.User{}, api.NewError(401, "USER_DISABLED", "用户已停用")
	}
	return user, nil
}

func (s *AuthService) Ready(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

type AuditService struct {
	repo *repository.SupportRepository
}

func NewAuditService(repo *repository.SupportRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) List(ctx context.Context, filter repository.AuditFilter) ([]model.AuditEvent, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	return s.repo.ListAudits(ctx, filter)
}
