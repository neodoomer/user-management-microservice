package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user-management-microservice/internal/apperr"
	"github.com/user-management-microservice/internal/db"
	"github.com/user-management-microservice/internal/dto"
	"github.com/user-management-microservice/pkg/hasher"
)

func newTestUserService(q *mockQuerier) *UserService {
	h := hasher.NewBcryptHasher()
	return NewUserService(q, h)
}

func TestUserService_CreateUser_Success(t *testing.T) {
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

	svc := newTestUserService(mq)
	resp, err := svc.CreateUser(context.Background(), dto.CreateUserRequest{
		Email:    "new@example.com",
		Username: "newuser",
		Password: "password123",
		Role:     "admin",
	})

	if err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}
	if resp.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", resp.Email, "new@example.com")
	}
	if resp.Role != "admin" {
		t.Errorf("Role = %q, want %q", resp.Role, "admin")
	}
}

func TestUserService_CreateUser_Conflict(t *testing.T) {
	mq := &mockQuerier{
		createUserFn: func(_ context.Context, _ db.CreateUserParams) (db.CreateUserRow, error) {
			return db.CreateUserRow{}, errors.New("unique violation")
		},
	}

	svc := newTestUserService(mq)
	_, err := svc.CreateUser(context.Background(), dto.CreateUserRequest{
		Email:    "dup@example.com",
		Username: "dupuser",
		Password: "password123",
		Role:     "user",
	})

	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}
}

func TestUserService_ListUsers_Success(t *testing.T) {
	uid1, uid2 := uuid.New(), uuid.New()
	now := nowTimestamptz()

	mq := &mockQuerier{
		listUsersFn: func(_ context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error) {
			return []db.ListUsersRow{
				{ID: toPgtypeUUID(uid1), Email: "a@example.com", Username: "usera", Role: "admin", CreatedAt: now, UpdatedAt: now},
				{ID: toPgtypeUUID(uid2), Email: "b@example.com", Username: "userb", Role: "user", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
		countUsersFn: func(_ context.Context) (int64, error) {
			return 2, nil
		},
	}

	svc := newTestUserService(mq)
	resp, err := svc.ListUsers(context.Background(), dto.ListUsersRequest{Page: 1, Limit: 10})

	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}
	if resp.Meta.Total != 2 {
		t.Errorf("Meta.Total = %d, want 2", resp.Meta.Total)
	}
	if resp.Meta.Page != 1 {
		t.Errorf("Meta.Page = %d, want 1", resp.Meta.Page)
	}
}

func TestUserService_ListUsers_DefaultPagination(t *testing.T) {
	var capturedParams db.ListUsersParams

	mq := &mockQuerier{
		listUsersFn: func(_ context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error) {
			capturedParams = arg
			return nil, nil
		},
		countUsersFn: func(_ context.Context) (int64, error) {
			return 0, nil
		},
	}

	svc := newTestUserService(mq)
	_, err := svc.ListUsers(context.Background(), dto.ListUsersRequest{Page: 0, Limit: 0})

	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if capturedParams.Limit != 20 {
		t.Errorf("default Limit = %d, want 20", capturedParams.Limit)
	}
	if capturedParams.Offset != 0 {
		t.Errorf("default Offset = %d, want 0", capturedParams.Offset)
	}
}

func TestUserService_ListUsers_DBError(t *testing.T) {
	mq := &mockQuerier{
		listUsersFn: func(_ context.Context, _ db.ListUsersParams) ([]db.ListUsersRow, error) {
			return nil, errors.New("db connection lost")
		},
	}

	svc := newTestUserService(mq)
	_, err := svc.ListUsers(context.Background(), dto.ListUsersRequest{Page: 1, Limit: 10})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func init() {
	_ = time.Now()
}
