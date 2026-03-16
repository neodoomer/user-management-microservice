package service

import (
	"context"
	"errors"
	"testing"
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

func newTestAuthService(q *mockQuerier) *AuthService {
	h := hasher.NewBcryptHasher()
	j := token.NewJWTManager("test-secret", 15*time.Minute)
	return NewAuthService(q, h, j)
}

func TestAuthService_SignUp_Success(t *testing.T) {
	uid := uuid.New()
	mq := &mockQuerier{
		createUserFn: func(_ context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
			return db.CreateUserRow{
				ID:        toPgtypeUUID(uid),
				Email:     arg.Email,
				Username:  arg.Username,
				Role:      arg.Role,
				CreatedAt: nowTimestamptz(),
				UpdatedAt: nowTimestamptz(),
			}, nil
		},
	}

	svc := newTestAuthService(mq)
	resp, err := svc.SignUp(context.Background(), dto.SignUpRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("SignUp() error: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("User.Email = %q, want %q", resp.User.Email, "test@example.com")
	}
	if resp.User.Role != "user" {
		t.Errorf("User.Role = %q, want %q", resp.User.Role, "user")
	}
}

func TestAuthService_SignUp_DuplicateEmail(t *testing.T) {
	mq := &mockQuerier{
		createUserFn: func(_ context.Context, _ db.CreateUserParams) (db.CreateUserRow, error) {
			return db.CreateUserRow{}, errors.New("duplicate key")
		},
	}

	svc := newTestAuthService(mq)
	_, err := svc.SignUp(context.Background(), dto.SignUpRequest{
		Email:    "dup@example.com",
		Username: "dupuser",
		Password: "password123",
	})

	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}
}

func TestAuthService_SignIn_Success(t *testing.T) {
	h := hasher.NewBcryptHasher()
	hashed, _ := h.Hash("password123")
	uid := uuid.New()

	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, email string) (db.User, error) {
			return db.User{
				ID:           toPgtypeUUID(uid),
				Email:        email,
				Username:     "testuser",
				PasswordHash: hashed,
				Role:         "admin",
				CreatedAt:    nowTimestamptz(),
				UpdatedAt:    nowTimestamptz(),
			}, nil
		},
	}

	svc := newTestAuthService(mq)
	resp, err := svc.SignIn(context.Background(), dto.SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("SignIn() error: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Role != "admin" {
		t.Errorf("User.Role = %q, want %q", resp.User.Role, "admin")
	}
}

func TestAuthService_SignIn_UserNotFound(t *testing.T) {
	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{}, pgx.ErrNoRows
		},
	}

	svc := newTestAuthService(mq)
	_, err := svc.SignIn(context.Background(), dto.SignInRequest{
		Email:    "noone@example.com",
		Password: "password123",
	})

	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestAuthService_SignIn_WrongPassword(t *testing.T) {
	h := hasher.NewBcryptHasher()
	hashed, _ := h.Hash("correctpassword")

	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{
				ID:           toPgtypeUUID(uuid.New()),
				PasswordHash: hashed,
				CreatedAt:    nowTimestamptz(),
				UpdatedAt:    nowTimestamptz(),
			}, nil
		},
	}

	svc := newTestAuthService(mq)
	_, err := svc.SignIn(context.Background(), dto.SignInRequest{
		Email:    "user@example.com",
		Password: "wrongpassword",
	})

	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestAuthService_ForgotPassword_ExistingUser(t *testing.T) {
	uid := uuid.New()
	var capturedParams db.CreatePasswordResetParams

	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{
				ID:        toPgtypeUUID(uid),
				Email:     "user@example.com",
				CreatedAt: nowTimestamptz(),
				UpdatedAt: nowTimestamptz(),
			}, nil
		},
		createPasswordResetFn: func(_ context.Context, arg db.CreatePasswordResetParams) (db.PasswordReset, error) {
			capturedParams = arg
			return db.PasswordReset{}, nil
		},
	}

	svc := newTestAuthService(mq)
	err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{
		Email: "user@example.com",
	})

	if err != nil {
		t.Fatalf("ForgotPassword() error: %v", err)
	}
	if capturedParams.TokenHash == "" {
		t.Error("expected non-empty token hash")
	}
}

func TestAuthService_ForgotPassword_NonExistentUser(t *testing.T) {
	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{}, pgx.ErrNoRows
		},
	}

	svc := newTestAuthService(mq)
	err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{
		Email: "nobody@example.com",
	})

	if err != nil {
		t.Fatalf("ForgotPassword() should succeed silently for missing user, got: %v", err)
	}
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	uid := uuid.New()
	resetID := uuid.New()
	var updatedHash string

	mq := &mockQuerier{
		getPasswordResetByTokenFn: func(_ context.Context, _ string) (db.PasswordReset, error) {
			return db.PasswordReset{
				ID:        toPgtypeUUID(resetID),
				UserID:    toPgtypeUUID(uid),
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(1 * time.Hour), Valid: true},
			}, nil
		},
		updatePasswordFn: func(_ context.Context, arg db.UpdatePasswordParams) error {
			updatedHash = arg.PasswordHash
			return nil
		},
		markPasswordResetUsedFn: func(_ context.Context, _ pgtype.UUID) error {
			return nil
		},
	}

	svc := newTestAuthService(mq)
	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:       "some-valid-token",
		NewPassword: "newpassword123",
	})

	if err != nil {
		t.Fatalf("ResetPassword() error: %v", err)
	}
	if updatedHash == "" {
		t.Error("expected password hash to be updated")
	}
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	mq := &mockQuerier{
		getPasswordResetByTokenFn: func(_ context.Context, _ string) (db.PasswordReset, error) {
			return db.PasswordReset{}, pgx.ErrNoRows
		},
	}

	svc := newTestAuthService(mq)
	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:       "bad-token",
		NewPassword: "newpassword123",
	})

	if !errors.Is(err, apperr.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got: %v", err)
	}
}

func TestAuthService_ResetPassword_ExpiredToken(t *testing.T) {
	mq := &mockQuerier{
		getPasswordResetByTokenFn: func(_ context.Context, _ string) (db.PasswordReset, error) {
			return db.PasswordReset{
				ID:        toPgtypeUUID(uuid.New()),
				UserID:    toPgtypeUUID(uuid.New()),
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
			}, nil
		},
	}

	svc := newTestAuthService(mq)
	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:       "expired-token",
		NewPassword: "newpassword123",
	})

	if !errors.Is(err, apperr.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got: %v", err)
	}
}
