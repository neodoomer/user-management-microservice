package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/user-management-microservice/internal/apperr"
	"github.com/user-management-microservice/internal/db"
	"github.com/user-management-microservice/internal/dto"
	"github.com/user-management-microservice/pkg/hasher"
	"github.com/user-management-microservice/pkg/token"
)

type AuthService struct {
	queries db.Querier
	hasher  hasher.PasswordHasher
	jwt     token.JWTManager
}

func NewAuthService(q db.Querier, h hasher.PasswordHasher, j token.JWTManager) *AuthService {
	return &AuthService{queries: q, hasher: h, jwt: j}
}

func (s *AuthService) SignUp(ctx context.Context, req dto.SignUpRequest) (*dto.AuthResponse, error) {
	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	row, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
		Role:         "user",
	})
	if err != nil {
		return nil, fmt.Errorf("%w: email or username already taken", apperr.ErrConflict)
	}

	uid := uuidFromPgtype(row.ID)
	tkn, err := s.jwt.Generate(uid, row.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &dto.AuthResponse{
		Token: tkn,
		User: dto.UserResponse{
			ID:        uid.String(),
			Email:     row.Email,
			Username:  row.Username,
			Role:      row.Role,
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		},
	}, nil
}

func (s *AuthService) SignIn(ctx context.Context, req dto.SignInRequest) (*dto.AuthResponse, error) {
	user, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: invalid credentials", apperr.ErrUnauthorized)
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := s.hasher.Compare(user.PasswordHash, req.Password); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", apperr.ErrUnauthorized)
	}

	uid := uuidFromPgtype(user.ID)
	tkn, err := s.jwt.Generate(uid, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &dto.AuthResponse{
		Token: tkn,
		User: dto.UserResponse{
			ID:        uid.String(),
			Email:     user.Email,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt.Time,
			UpdatedAt: user.UpdatedAt.Time,
		},
	}, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	user, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get user: %w", err)
	}

	rawToken, err := generateRandomToken(32)
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = s.queries.CreatePasswordReset(ctx, db.CreatePasswordResetParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create password reset: %w", err)
	}

	log.Printf("[RESET TOKEN] user=%s token=%s (expires=%s)", user.Email, rawToken, expiresAt.Format(time.RFC3339))

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	tokenHash := hashToken(req.Token)

	reset, err := s.queries.GetPasswordResetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: invalid or expired reset token", apperr.ErrBadRequest)
		}
		return fmt.Errorf("get password reset: %w", err)
	}

	if time.Now().After(reset.ExpiresAt.Time) {
		return fmt.Errorf("%w: reset token has expired", apperr.ErrBadRequest)
	}

	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.queries.UpdatePassword(ctx, db.UpdatePasswordParams{
		ID:           reset.UserID,
		PasswordHash: hash,
	}); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := s.queries.MarkPasswordResetUsed(ctx, reset.ID); err != nil {
		return fmt.Errorf("mark reset used: %w", err)
	}

	return nil
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func uuidFromPgtype(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}
