package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"micropanel/internal/config"
	"micropanel/internal/database"
	"micropanel/internal/handlers"
	"micropanel/internal/middleware"
	"micropanel/internal/repository"
	"micropanel/internal/services"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  "Start the MicroPanel web server and API.",
	Run:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	siteRepo := repository.NewSiteRepository(db)
	domainRepo := repository.NewDomainRepository(db)
	deployRepo := repository.NewDeployRepository(db)
	redirectRepo := repository.NewRedirectRepository(db)
	authZoneRepo := repository.NewAuthZoneRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	auditService := services.NewAuditService(auditRepo)
	authService := services.NewAuthService(userRepo, sessionRepo)
	siteService := services.NewSiteService(siteRepo, domainRepo, cfg)
	nginxService := services.NewNginxService(cfg, siteRepo, domainRepo)
	nginxService.SetRedirectRepo(redirectRepo)
	nginxService.SetAuthZoneRepo(authZoneRepo)
	deployService := services.NewDeployService(cfg, deployRepo, siteRepo)
	sslService := services.NewSSLService(cfg, domainRepo, nginxService)
	redirectService := services.NewRedirectService(redirectRepo, nginxService)
	authZoneService := services.NewAuthZoneService(cfg, authZoneRepo, nginxService)
	fileService := services.NewFileService(cfg)

	authHandler := handlers.NewAuthHandler(authService, auditService)
	siteHandler := handlers.NewSiteHandler(siteService, deployService, redirectService, authZoneService, auditService)
	domainHandler := handlers.NewDomainHandler(domainRepo, siteService, nginxService, auditService)
	deployHandler := handlers.NewDeployHandler(deployService, siteService, auditService)
	sslHandler := handlers.NewSSLHandler(sslService, siteService, auditService)
	redirectHandler := handlers.NewRedirectHandler(redirectService, siteService, auditService)
	authZoneHandler := handlers.NewAuthZoneHandler(authZoneService, siteService, auditService)
	fileHandler := handlers.NewFileHandler(fileService, siteService, auditService)
	auditHandler := handlers.NewAuditHandler(auditService, userRepo)
	userHandler := handlers.NewUserHandler(userRepo, auditService)

	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	loginLimiter := middleware.NewRateLimiter(5, time.Minute)
	apiLimiter := middleware.NewRateLimiter(100, time.Minute)

	r.Use(middleware.CSRF())
	r.Static("/static", "./web/static")

	r.GET("/login", authHandler.LoginPage)
	r.POST("/login", loginLimiter.Middleware(), authHandler.Login)

	protected := r.Group("/")
	protected.Use(middleware.Auth(authService))
	protected.Use(apiLimiter.Middleware())
	{
		protected.POST("/logout", authHandler.Logout)
		protected.GET("/", siteHandler.Dashboard)
		protected.POST("/sites", siteHandler.Create)
		protected.GET("/sites/:id", siteHandler.View)
		protected.GET("/sites/:id/files-page", siteHandler.Files)
		protected.POST("/sites/:id", siteHandler.Update)
		protected.DELETE("/sites/:id", siteHandler.Delete)

		protected.POST("/sites/:id/domains", domainHandler.Create)
		protected.DELETE("/sites/:id/domains/:domainId", domainHandler.Delete)
		protected.POST("/sites/:id/domains/:domainId/primary", domainHandler.SetPrimary)

		protected.POST("/sites/:id/deploy", deployHandler.Upload)
		protected.POST("/sites/:id/rollback", deployHandler.Rollback)

		protected.POST("/sites/:id/ssl/issue", sslHandler.Issue)
		protected.POST("/ssl/renew", sslHandler.Renew)

		protected.POST("/sites/:id/redirects", redirectHandler.Create)
		protected.POST("/sites/:id/redirects/:redirectId", redirectHandler.Update)
		protected.DELETE("/sites/:id/redirects/:redirectId", redirectHandler.Delete)
		protected.POST("/sites/:id/redirects/:redirectId/toggle", redirectHandler.Toggle)

		protected.POST("/sites/:id/auth-zones", authZoneHandler.Create)
		protected.POST("/sites/:id/auth-zones/:zoneId", authZoneHandler.Update)
		protected.DELETE("/sites/:id/auth-zones/:zoneId", authZoneHandler.Delete)
		protected.POST("/sites/:id/auth-zones/:zoneId/toggle", authZoneHandler.Toggle)
		protected.POST("/sites/:id/auth-zones/:zoneId/users", authZoneHandler.CreateUser)
		protected.DELETE("/sites/:id/auth-zones/:zoneId/users/:userId", authZoneHandler.DeleteUser)

		protected.GET("/sites/:id/files", fileHandler.List)
		protected.GET("/sites/:id/files/read", fileHandler.Read)
		protected.POST("/sites/:id/files/write", fileHandler.Write)
		protected.POST("/sites/:id/files/create", fileHandler.Create)
		protected.DELETE("/sites/:id/files", fileHandler.Delete)
		protected.POST("/sites/:id/files/rename", fileHandler.Rename)
		protected.POST("/sites/:id/files/upload", fileHandler.Upload)
		protected.GET("/sites/:id/files/download", fileHandler.Download)
		protected.GET("/sites/:id/files/preview", fileHandler.Preview)
		protected.GET("/sites/:id/files/info", fileHandler.Info)

		protected.GET("/audit", auditHandler.List)
		protected.GET("/api/audit", auditHandler.ListAPI)

		protected.GET("/users", userHandler.List)
		protected.POST("/users", userHandler.Create)
		protected.POST("/users/:id", userHandler.Update)
		protected.POST("/users/:id/toggle", userHandler.ToggleActive)
		protected.DELETE("/users/:id", userHandler.Delete)

		protected.GET("/profile", userHandler.Profile)
		protected.POST("/profile/password", userHandler.ChangePassword)
	}

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
