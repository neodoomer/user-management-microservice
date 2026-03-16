package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/user-management-microservice/pkg/token"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	mgr := token.NewJWTManager("test-secret", 15*time.Minute)
	uid := uuid.New()
	tokenStr, _ := mgr.Generate(uid, "admin")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(mgr)(func(c echo.Context) error {
		gotUID := c.Get(ContextKeyUserID).(string)
		gotRole := c.Get(ContextKeyRole).(string)

		if gotUID != uid.String() {
			t.Errorf("UserID = %q, want %q", gotUID, uid.String())
		}
		if gotRole != "admin" {
			t.Errorf("Role = %q, want %q", gotRole, "admin")
		}
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	mgr := token.NewJWTManager("test-secret", 15*time.Minute)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(mgr)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", he.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	mgr := token.NewJWTManager("test-secret", 15*time.Minute)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic some-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(mgr)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", he.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	mgr := token.NewJWTManager("test-secret", -1*time.Second)
	tokenStr, _ := mgr.Generate(uuid.New(), "user")

	mgrValidate := token.NewJWTManager("test-secret", 15*time.Minute)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(mgrValidate)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", he.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole_Allowed(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextKeyRole, "admin")

	handler := RequireRole("admin", "superadmin")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_Denied(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextKeyRole, "user")

	handler := RequireRole("admin")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", he.Code, http.StatusForbidden)
	}
}

func TestRequireRole_NoRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := RequireRole("admin")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", he.Code, http.StatusForbidden)
	}
}
