package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/user-management-microservice/internal/apperr"
	"github.com/user-management-microservice/internal/dto"
)

func GlobalErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var he *echo.HTTPError
	if errors.As(err, &he) {
		msg, _ := he.Message.(string)
		if msg == "" {
			if details, ok := he.Message.([]string); ok {
				_ = c.JSON(he.Code, dto.ErrorResponse{Error: "validation failed", Details: details})
				return
			}
		}
		_ = c.JSON(he.Code, dto.ErrorResponse{Error: msg})
		return
	}

	code := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, apperr.ErrNotFound):
		code = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, apperr.ErrConflict):
		code = http.StatusConflict
		message = err.Error()
	case errors.Is(err, apperr.ErrUnauthorized):
		code = http.StatusUnauthorized
		message = err.Error()
	case errors.Is(err, apperr.ErrForbidden):
		code = http.StatusForbidden
		message = err.Error()
	case errors.Is(err, apperr.ErrBadRequest):
		code = http.StatusBadRequest
		message = err.Error()
	}

	_ = c.JSON(code, dto.ErrorResponse{Error: message})
}
