package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"micropanel/internal/config"
	"micropanel/internal/database"
	"micropanel/internal/handlers"
	"micropanel/internal/middleware"
	"micropanel/internal/repository"
	"micropanel/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	siteRepo := repository.NewSiteRepository(db)
	domainRepo := repository.NewDomainRepository(db)
	deployRepo := repository.NewDeployRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, sessionRepo)
	siteService := services.NewSiteService(siteRepo, domainRepo, cfg)
	nginxService := services.NewNginxService(cfg, siteRepo, domainRepo)
	deployService := services.NewDeployService(cfg, deployRepo, siteRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	siteHandler := handlers.NewSiteHandler(siteService, deployService)
	domainHandler := handlers.NewDomainHandler(domainRepo, siteService, nginxService)
	deployHandler := handlers.NewDeployHandler(deployService, siteService)

	// Setup Gin
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Global middleware
	r.Use(middleware.CSRF())

	// Static files
	r.Static("/static", "./web/static")

	// Public routes
	r.GET("/login", authHandler.LoginPage)
	r.POST("/login", authHandler.Login)

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.Auth(authService))
	{
		protected.POST("/logout", authHandler.Logout)
		protected.GET("/", siteHandler.Dashboard)
		protected.POST("/sites", siteHandler.Create)
		protected.GET("/sites/:id", siteHandler.View)
		protected.POST("/sites/:id", siteHandler.Update)
		protected.DELETE("/sites/:id", siteHandler.Delete)

		// Domain routes
		protected.POST("/sites/:id/domains", domainHandler.Create)
		protected.DELETE("/sites/:id/domains/:domainId", domainHandler.Delete)
		protected.POST("/sites/:id/domains/:domainId/primary", domainHandler.SetPrimary)

		// Deploy routes
		protected.POST("/sites/:id/deploy", deployHandler.Upload)
		protected.POST("/sites/:id/rollback", deployHandler.Rollback)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		log.Printf("Starting server on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}
