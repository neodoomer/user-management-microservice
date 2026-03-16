package service

import (
	"context"
	"fmt"

	"github.com/user-management-microservice/internal/apperr"
	"github.com/user-management-microservice/internal/db"
	"github.com/user-management-microservice/internal/dto"
	"github.com/user-management-microservice/pkg/hasher"
)

type UserService struct {
	queries db.Querier
	hasher  hasher.PasswordHasher
}

func NewUserService(q db.Querier, h hasher.PasswordHasher) *UserService {
	return &UserService{queries: q, hasher: h}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	row, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
		Role:         req.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: email or username already taken", apperr.ErrConflict)
	}

	uid := uuidFromPgtype(row.ID)
	return &dto.UserResponse{
		ID:        uid.String(),
		Email:     row.Email,
		Username:  row.Username,
		Role:      row.Role,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req dto.ListUsersRequest) (*dto.ListUsersResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}

	offset := (req.Page - 1) * req.Limit

	rows, err := s.queries.ListUsers(ctx, db.ListUsersParams{
		Limit:  int32(req.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	total, err := s.queries.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	users := make([]dto.UserResponse, 0, len(rows))
	for _, r := range rows {
		uid := uuidFromPgtype(r.ID)
		users = append(users, dto.UserResponse{
			ID:        uid.String(),
			Email:     r.Email,
			Username:  r.Username,
			Role:      r.Role,
			CreatedAt: r.CreatedAt.Time,
			UpdatedAt: r.UpdatedAt.Time,
		})
	}

	return &dto.ListUsersResponse{
		Data: users,
		Meta: dto.PaginationMeta{
			Page:  req.Page,
			Limit: req.Limit,
			Total: total,
		},
	}, nil
}
