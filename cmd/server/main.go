package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/user-management-microservice/internal/config"
	"github.com/user-management-microservice/internal/db"
	"github.com/user-management-microservice/internal/handler"
	"github.com/user-management-microservice/internal/service"
	"github.com/user-management-microservice/pkg/hasher"
	"github.com/user-management-microservice/pkg/token"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn := cfg.DB.DSN()

	runMigrations(dsn)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to database")

	queries := db.New(pool)
	pwHasher := hasher.NewBcryptHasher()
	jwtManager := token.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry)

	authService := service.NewAuthService(queries, pwHasher, jwtManager)
	userService := service.NewUserService(queries, pwHasher)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	e := echo.New()
	e.HideBanner = true
	e.Validator = handler.NewCustomValidator()
	e.HTTPErrorHandler = handler.GlobalErrorHandler

	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	handler.RegisterRoutes(e, authHandler, userHandler, jwtManager)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	go func() {
		log.Printf("starting server on %s", addr)
		if err := e.Start(addr); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	log.Println("server exited cleanly")
}

func runMigrations(dsn string) {
	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		m, err = migrate.New("file:///migrations", dsn)
		if err != nil {
			log.Fatalf("failed to create migrate instance: %v", err)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("migrations applied successfully")
}
