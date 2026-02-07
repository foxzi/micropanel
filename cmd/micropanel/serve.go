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
	"micropanel/internal/models"
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

	// Try multiple migration paths
	migrationPaths := []string{
		"/usr/share/micropanel/migrations",
		"migrations",
	}
	var migrated bool
	for _, path := range migrationPaths {
		if _, err := os.Stat(path); err == nil {
			if err := db.Migrate(path); err != nil {
				log.Fatalf("Failed to run migrations: %v", err)
			}
			migrated = true
			break
		}
	}
	if !migrated {
		log.Fatal("Migrations directory not found")
	}

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	siteRepo := repository.NewSiteRepository(db)

	// Startup validation
	if err := validateStartup(cfg, userRepo); err != nil {
		log.Fatal(err)
	}
	domainRepo := repository.NewDomainRepository(db)
	deployRepo := repository.NewDeployRepository(db)
	redirectRepo := repository.NewRedirectRepository(db)
	authZoneRepo := repository.NewAuthZoneRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	auditService := services.NewAuditService(auditRepo)
	authService := services.NewAuthService(userRepo, sessionRepo)
	settingsService := services.NewSettingsService(settingsRepo)
	go settingsService.FetchExternalIP()
	siteService := services.NewSiteService(siteRepo, domainRepo, cfg)
	nginxService := services.NewNginxService(cfg, siteRepo, domainRepo)
	nginxService.SetRedirectRepo(redirectRepo)
	nginxService.SetAuthZoneRepo(authZoneRepo)
	deployService := services.NewDeployService(cfg, deployRepo, siteRepo)
	sslService := services.NewSSLService(cfg, siteRepo, domainRepo, nginxService)
	redirectService := services.NewRedirectService(redirectRepo, nginxService)
	authZoneService := services.NewAuthZoneService(cfg, authZoneRepo, nginxService)
	fileService := services.NewFileService(cfg)

	authHandler := handlers.NewAuthHandler(authService, auditService)
	siteHandler := handlers.NewSiteHandler(siteService, deployService, redirectService, authZoneService, auditService, settingsService)
	domainHandler := handlers.NewDomainHandler(domainRepo, siteService, nginxService, auditService)
	settingsHandler := handlers.NewSettingsHandler(settingsService, auditService)
	deployHandler := handlers.NewDeployHandler(deployService, siteService, auditService)
	sslHandler := handlers.NewSSLHandler(sslService, siteService, auditService)
	redirectHandler := handlers.NewRedirectHandler(redirectService, siteService, auditService)
	authZoneHandler := handlers.NewAuthZoneHandler(authZoneService, siteService, auditService)
	fileHandler := handlers.NewFileHandler(fileService, siteService, auditService)
	auditHandler := handlers.NewAuditHandler(auditService, userRepo)
	userHandler := handlers.NewUserHandler(userRepo, auditService)
	apiHandler := handlers.NewAPIHandler(siteService, deployService, nginxService, auditService)

	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	loginLimiter := middleware.NewRateLimiter(5, time.Minute)
	apiLimiter := middleware.NewRateLimiter(100, time.Minute)

	// Panel routes with IP whitelist
	panelGroup := r.Group("/")
	panelGroup.Use(middleware.IPWhitelist(cfg.Security.PanelAllowedIPs))
	panelGroup.Use(middleware.CSRF())

	// Try multiple static paths
	staticPaths := []string{
		"/usr/share/micropanel/web/static",
		"web/static",
	}
	for _, path := range staticPaths {
		if _, err := os.Stat(path); err == nil {
			panelGroup.Static("/static", path)
			break
		}
	}

	panelGroup.GET("/login", authHandler.LoginPage)
	panelGroup.POST("/login", loginLimiter.Middleware(), authHandler.Login)

	protected := panelGroup.Group("/")
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

		protected.GET("/settings", settingsHandler.Page)
		protected.POST("/settings", settingsHandler.Update)

		protected.GET("/users", userHandler.List)
		protected.POST("/users", userHandler.Create)
		protected.POST("/users/:id", userHandler.Update)
		protected.POST("/users/:id/toggle", userHandler.ToggleActive)
		protected.DELETE("/users/:id", userHandler.Delete)

		protected.GET("/profile", userHandler.Profile)
		protected.POST("/profile/password", userHandler.ChangePassword)
	}

	// API routes
	if cfg.API.Enabled {
		apiGroup := r.Group("/api/v1")
		apiGroup.Use(middleware.IPWhitelist(cfg.Security.APIAllowedIPs))
		apiGroup.Use(middleware.APIToken(cfg.API.Tokens))
		apiGroup.Use(apiLimiter.Middleware())
		{
			apiGroup.POST("/sites", apiHandler.CreateSite)
			apiGroup.GET("/sites", apiHandler.ListSites)
			apiGroup.GET("/sites/:id", apiHandler.GetSite)
			apiGroup.DELETE("/sites/:id", apiHandler.DeleteSite)
			apiGroup.POST("/sites/:id/deploy", apiHandler.Deploy)
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
		log.Printf("Starting server on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}

func validateStartup(cfg *config.Config, userRepo *repository.UserRepository) error {
	var errors []string

	// Check panel_domain
	if cfg.App.PanelDomain == "" {
		errors = append(errors, "panel_domain is not configured")
	}

	// Check admin user exists
	users, err := userRepo.List()
	if err != nil {
		return fmt.Errorf("failed to check users: %v", err)
	}

	hasAdmin := false
	for _, u := range users {
		if u.Role == models.RoleAdmin && u.IsActive {
			hasAdmin = true
			break
		}
	}

	if !hasAdmin {
		errors = append(errors, "no admin user found")
	}

	if len(errors) > 0 {
		log.Println("")
		log.Println("========================================")
		log.Println("  MicroPanel configuration incomplete!")
		log.Println("========================================")
		log.Println("")
		for _, e := range errors {
			log.Printf("  - %s", e)
		}
		log.Println("")
		log.Println("To fix:")
		log.Println("")
		if cfg.App.PanelDomain == "" {
			log.Println("  1. Set panel_domain in /etc/micropanel/config.yaml")
			log.Println("     Example: panel_domain: panel.example.com")
			log.Println("")
		}
		if !hasAdmin {
			log.Println("  2. Create admin user:")
			log.Println("     micropanel user create -e admin@example.com -p yourpassword -r admin")
			log.Println("")
		}
		log.Println("  3. Setup nginx:")
		log.Println("     /usr/share/micropanel/scripts/setup-panel-nginx.sh")
		log.Println("")
		log.Println("  4. Start service:")
		log.Println("     systemctl start micropanel")
		log.Println("")
		return fmt.Errorf("startup validation failed")
	}

	return nil
}
