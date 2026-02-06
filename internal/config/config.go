package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Sites    SitesConfig    `yaml:"sites"`
	Nginx    NginxConfig    `yaml:"nginx"`
	SSL      SSLConfig      `yaml:"ssl"`
	Limits   LimitsConfig   `yaml:"limits"`
}

type LimitsConfig struct {
	MaxZipSize      int64 `yaml:"max_zip_size"`       // bytes
	MaxFileSize     int64 `yaml:"max_file_size"`      // bytes
	MaxUploadSize   int64 `yaml:"max_upload_size"`    // bytes
	MaxSitesPerUser int   `yaml:"max_sites_per_user"` // 0 = unlimited
}

type SSLConfig struct {
	Email   string `yaml:"email"`
	Staging bool   `yaml:"staging"`
}

type AppConfig struct {
	Env    string `yaml:"env"`
	Port   int    `yaml:"port"`
	Secret string `yaml:"secret"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type SitesConfig struct {
	Path string `yaml:"path"`
}

type NginxConfig struct {
	ConfigPath string `yaml:"config_path"`
	ReloadCmd  string `yaml:"reload_cmd"`
}

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Env:    "development",
			Port:   8080,
			Secret: "change-me-in-production",
		},
		Database: DatabaseConfig{
			Path: "./data/micropanel.db",
		},
		Sites: SitesConfig{
			Path: "/var/www/panel/sites",
		},
		Nginx: NginxConfig{
			ConfigPath: "/etc/nginx/sites-enabled",
			ReloadCmd:  "nginx -s reload",
		},
		SSL: SSLConfig{
			Email:   "",
			Staging: false,
		},
		Limits: LimitsConfig{
			MaxZipSize:      100 * 1024 * 1024, // 100MB
			MaxFileSize:     5 * 1024 * 1024,   // 5MB
			MaxUploadSize:   10 * 1024 * 1024,  // 10MB
			MaxSitesPerUser: 0,                 // unlimited
		},
	}

	// Load from YAML if exists
	if data, err := os.ReadFile("config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.App.Env = env
	}
	if port := os.Getenv("APP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.App.Port = p
		}
	}
	if secret := os.Getenv("APP_SECRET"); secret != "" {
		cfg.App.Secret = secret
	}
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if sitesPath := os.Getenv("SITES_PATH"); sitesPath != "" {
		cfg.Sites.Path = sitesPath
	}
	if nginxPath := os.Getenv("NGINX_CONFIG_PATH"); nginxPath != "" {
		cfg.Nginx.ConfigPath = nginxPath
	}
	if sslEmail := os.Getenv("SSL_EMAIL"); sslEmail != "" {
		cfg.SSL.Email = sslEmail
	}
	if sslStaging := os.Getenv("SSL_STAGING"); sslStaging == "true" {
		cfg.SSL.Staging = true
	}
	if maxZip := os.Getenv("MAX_ZIP_SIZE"); maxZip != "" {
		if v, err := strconv.ParseInt(maxZip, 10, 64); err == nil {
			cfg.Limits.MaxZipSize = v
		}
	}
	if maxFile := os.Getenv("MAX_FILE_SIZE"); maxFile != "" {
		if v, err := strconv.ParseInt(maxFile, 10, 64); err == nil {
			cfg.Limits.MaxFileSize = v
		}
	}
	if maxUpload := os.Getenv("MAX_UPLOAD_SIZE"); maxUpload != "" {
		if v, err := strconv.ParseInt(maxUpload, 10, 64); err == nil {
			cfg.Limits.MaxUploadSize = v
		}
	}
	if maxSites := os.Getenv("MAX_SITES_PER_USER"); maxSites != "" {
		if v, err := strconv.Atoi(maxSites); err == nil {
			cfg.Limits.MaxSitesPerUser = v
		}
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}
