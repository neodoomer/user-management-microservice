package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/user-management-microservice/internal/dto"
	"github.com/user-management-microservice/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(us *service.UserService) *UserHandler {
	return &UserHandler{userService: us}
}

func (h *UserHandler) CreateUser(c echo.Context) error {
	var req dto.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	resp, err := h.userService.CreateUser(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *UserHandler) ListUsers(c echo.Context) error {
	var req dto.ListUsersRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid query parameters")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	resp, err := h.userService.ListUsers(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}
