package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/user-management-microservice/internal/db"
)

type mockQuerier struct {
	createUserFn              func(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error)
	getUserByEmailFn          func(ctx context.Context, email string) (db.User, error)
	getUserByIDFn             func(ctx context.Context, id pgtype.UUID) (db.User, error)
	listUsersFn               func(ctx context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error)
	countUsersFn              func(ctx context.Context) (int64, error)
	updatePasswordFn          func(ctx context.Context, arg db.UpdatePasswordParams) error
	createPasswordResetFn     func(ctx context.Context, arg db.CreatePasswordResetParams) (db.PasswordReset, error)
	getPasswordResetByTokenFn func(ctx context.Context, tokenHash string) (db.PasswordReset, error)
	markPasswordResetUsedFn   func(ctx context.Context, id pgtype.UUID) error
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, arg)
	}
	return db.CreateUserRow{}, nil
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return db.User{}, nil
}

func (m *mockQuerier) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, id)
	}
	return db.User{}, nil
}

func (m *mockQuerier) ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) CountUsers(ctx context.Context) (int64, error) {
	if m.countUsersFn != nil {
		return m.countUsersFn(ctx)
	}
	return 0, nil
}

func (m *mockQuerier) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, arg)
	}
	return nil
}

func (m *mockQuerier) CreatePasswordReset(ctx context.Context, arg db.CreatePasswordResetParams) (db.PasswordReset, error) {
	if m.createPasswordResetFn != nil {
		return m.createPasswordResetFn(ctx, arg)
	}
	return db.PasswordReset{}, nil
}

func (m *mockQuerier) GetPasswordResetByTokenHash(ctx context.Context, tokenHash string) (db.PasswordReset, error) {
	if m.getPasswordResetByTokenFn != nil {
		return m.getPasswordResetByTokenFn(ctx, tokenHash)
	}
	return db.PasswordReset{}, nil
}

func (m *mockQuerier) MarkPasswordResetUsed(ctx context.Context, id pgtype.UUID) error {
	if m.markPasswordResetUsedFn != nil {
		return m.markPasswordResetUsedFn(ctx, id)
	}
	return nil
}

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func nowTimestamptz() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now(), Valid: true}
}
