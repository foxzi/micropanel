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

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}
