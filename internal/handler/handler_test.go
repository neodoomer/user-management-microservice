package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/user-management-microservice/internal/db"
	"github.com/user-management-microservice/internal/dto"
	"github.com/user-management-microservice/internal/service"
	"github.com/user-management-microservice/pkg/hasher"
	"github.com/user-management-microservice/pkg/token"
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

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = NewCustomValidator()
	e.HTTPErrorHandler = GlobalErrorHandler
	return e
}

// --- Auth Handler Tests ---

func TestAuthHandler_SignUp_Success(t *testing.T) {
	uid := uuid.New()
	mq := &mockQuerier{
		createUserFn: func(_ context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
			return db.CreateUserRow{
				ID: toPgtypeUUID(uid), Email: arg.Email, Username: arg.Username,
				Role: "user", CreatedAt: nowTimestamptz(), UpdatedAt: nowTimestamptz(),
			}, nil
		},
	}
	h := hasher.NewBcryptHasher()
	j := token.NewJWTManager("test-secret", 15*time.Minute)
	authSvc := service.NewAuthService(mq, h, j)
	ah := NewAuthHandler(authSvc)

	e := newTestEcho()
	body := `{"email":"test@example.com","username":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := ah.SignUp(c); err != nil {
		t.Fatalf("SignUp() error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthHandler_SignUp_ValidationError(t *testing.T) {
	mq := &mockQuerier{}
	h := hasher.NewBcryptHasher()
	j := token.NewJWTManager("test-secret", 15*time.Minute)
	authSvc := service.NewAuthService(mq, h, j)
	ah := NewAuthHandler(authSvc)

	e := newTestEcho()
	body := `{"email":"not-an-email","username":"ab","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ah.SignUp(c)
	if err == nil {
		t.Fatal("expected validation error")
	}

	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", he.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_SignIn_Success(t *testing.T) {
	h := hasher.NewBcryptHasher()
	hashed, _ := h.Hash("password123")
	uid := uuid.New()

	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, email string) (db.User, error) {
			return db.User{
				ID: toPgtypeUUID(uid), Email: email, Username: "testuser",
				PasswordHash: hashed, Role: "admin",
				CreatedAt: nowTimestamptz(), UpdatedAt: nowTimestamptz(),
			}, nil
		},
	}
	j := token.NewJWTManager("test-secret", 15*time.Minute)
	authSvc := service.NewAuthService(mq, h, j)
	ah := NewAuthHandler(authSvc)

	e := newTestEcho()
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := ah.SignIn(c); err != nil {
		t.Fatalf("SignIn() error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_SignIn_InvalidCredentials(t *testing.T) {
	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{}, pgx.ErrNoRows
		},
	}
	h := hasher.NewBcryptHasher()
	j := token.NewJWTManager("test-secret", 15*time.Minute)
	authSvc := service.NewAuthService(mq, h, j)
	ah := NewAuthHandler(authSvc)

	e := newTestEcho()
	body := `{"email":"wrong@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ah.SignIn(c)
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	mq := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{
				ID: toPgtypeUUID(uuid.New()), Email: "user@example.com",
				CreatedAt: nowTimestamptz(), UpdatedAt: nowTimestamptz(),
			}, nil
		},
		createPasswordResetFn: func(_ context.Context, _ db.CreatePasswordResetParams) (db.PasswordReset, error) {
			return db.PasswordReset{}, nil
		},
	}
	h := hasher.NewBcryptHasher()
	j := token.NewJWTManager("test-secret", 15*time.Minute)
	authSvc := service.NewAuthService(mq, h, j)
	ah := NewAuthHandler(authSvc)

	e := newTestEcho()
	body := `{"email":"user@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := ah.ForgotPassword(c); err != nil {
		t.Fatalf("ForgotPassword() error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// --- User Handler Tests ---

func TestUserHandler_CreateUser_Success(t *testing.T) {
	uid := uuid.New()
	mq := &mockQuerier{
		createUserFn: func(_ context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
			return db.CreateUserRow{
				ID: toPgtypeUUID(uid), Email: arg.Email, Username: arg.Username,
				Role: arg.Role, CreatedAt: nowTimestamptz(), UpdatedAt: nowTimestamptz(),
			}, nil
		},
	}
	h := hasher.NewBcryptHasher()
	userSvc := service.NewUserService(mq, h)
	uh := NewUserHandler(userSvc)

	e := newTestEcho()
	body := `{"email":"new@example.com","username":"newuser","password":"password123","role":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := uh.CreateUser(c); err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestUserHandler_ListUsers_Success(t *testing.T) {
	now := nowTimestamptz()
	mq := &mockQuerier{
		listUsersFn: func(_ context.Context, _ db.ListUsersParams) ([]db.ListUsersRow, error) {
			return []db.ListUsersRow{
				{ID: toPgtypeUUID(uuid.New()), Email: "a@example.com", Username: "usera", Role: "admin", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
		countUsersFn: func(_ context.Context) (int64, error) {
			return 1, nil
		},
	}
	h := hasher.NewBcryptHasher()
	userSvc := service.NewUserService(mq, h)
	uh := NewUserHandler(userSvc)

	e := newTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := uh.ListUsers(c); err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp dto.ListUsersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(resp.Data))
	}
}

// --- Error Handler Tests ---

func TestGlobalErrorHandler_AppError(t *testing.T) {
	e := newTestEcho()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"echo http error", echo.NewHTTPError(http.StatusTeapot, "teapot"), http.StatusTeapot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			GlobalErrorHandler(tt.err, c)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// --- Validator Tests ---

func TestCustomValidator_Valid(t *testing.T) {
	v := NewCustomValidator()

	req := dto.SignInRequest{Email: "test@example.com", Password: "password123"}
	if err := v.Validate(&req); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}

func TestCustomValidator_Invalid(t *testing.T) {
	v := NewCustomValidator()

	req := dto.SignInRequest{Email: "not-email", Password: ""}
	err := v.Validate(&req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", he.Code, http.StatusBadRequest)
	}
}
