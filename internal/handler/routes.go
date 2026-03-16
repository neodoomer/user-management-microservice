package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/user-management-microservice/internal/middleware"
	"github.com/user-management-microservice/pkg/token"
)

func RegisterRoutes(e *echo.Echo, ah *AuthHandler, uh *UserHandler, jwt token.JWTManager) {
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := e.Group("/api/v1")

	auth := v1.Group("/auth")
	auth.POST("/signup", ah.SignUp)
	auth.POST("/signin", ah.SignIn)
	auth.POST("/forgot-password", ah.ForgotPassword)
	auth.POST("/reset-password", ah.ResetPassword)

	users := v1.Group("/users",
		middleware.JWTAuth(jwt),
		middleware.RequireRole("admin"),
	)
	users.POST("", uh.CreateUser)
	users.GET("", uh.ListUsers)
}
