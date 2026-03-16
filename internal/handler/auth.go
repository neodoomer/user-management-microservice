package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/user-management-microservice/internal/dto"
	"github.com/user-management-microservice/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(as *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

func (h *AuthHandler) SignUp(c echo.Context) error {
	var req dto.SignUpRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	resp, err := h.authService.SignUp(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) SignIn(c echo.Context) error {
	var req dto.SignInRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	resp, err := h.authService.SignIn(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	if err := h.authService.ForgotPassword(c.Request().Context(), req); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "if the email exists, a reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	if err := h.authService.ResetPassword(c.Request().Context(), req); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "password has been reset successfully",
	})
}
