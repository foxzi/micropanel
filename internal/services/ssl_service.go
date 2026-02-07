package services

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"micropanel/internal/config"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

var (
	ErrCertbotFailed  = errors.New("certbot command failed")
	ErrNoDomains      = errors.New("no domains configured for site")
	ErrCertNotFound   = errors.New("certificate not found")
	ErrDomainNotFound = errors.New("domain not found")
)

type SSLService struct {
	config       *config.Config
	siteRepo     *repository.SiteRepository
	domainRepo   *repository.DomainRepository
	nginxService *NginxService
}

func NewSSLService(cfg *config.Config, siteRepo *repository.SiteRepository, domainRepo *repository.DomainRepository, nginxService *NginxService) *SSLService {
	return &SSLService{
		config:       cfg,
		siteRepo:     siteRepo,
		domainRepo:   domainRepo,
		nginxService: nginxService,
	}
}

// IssueCertificate requests a new SSL certificate for site (primary domain + www + aliases)
func (s *SSLService) IssueCertificate(siteID int64) error {
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return fmt.Errorf("get site: %w", err)
	}

	// Load aliases
	aliases, err := s.domainRepo.ListBySite(siteID)
	if err != nil {
		return fmt.Errorf("list aliases: %w", err)
	}
	site.Aliases = make([]models.Domain, len(aliases))
	for i, d := range aliases {
		site.Aliases[i] = *d
	}

	// Get all hostnames for certificate
	hostnames := site.GetAllHostnames()
	if len(hostnames) == 0 {
		return ErrNoDomains
	}

	// Build domain args for certbot
	var domainArgs []string
	for _, h := range hostnames {
		domainArgs = append(domainArgs, "-d", h)
	}

	// Run certbot (cert-name is the primary domain)
	args := []string{
		"certonly",
		"--webroot",
		"--webroot-path", "/var/www/certbot",
		"--email", s.config.SSL.Email,
		"--agree-tos",
		"--no-eff-email",
		"--non-interactive",
		"--cert-name", site.Name,
	}

	if s.config.SSL.Staging {
		args = append(args, "--staging")
	}

	args = append(args, domainArgs...)

	cmd := exec.Command("sudo", append([]string{"certbot"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	// Update site SSL status
	expiresAt, _ := s.GetCertificateExpiry(site.Name)
	site.SSLEnabled = true
	site.SSLExpiresAt = expiresAt
	if err := s.siteRepo.Update(site); err != nil {
		return fmt.Errorf("update site SSL status: %w", err)
	}

	// Regenerate nginx config with SSL
	if err := s.nginxService.ApplyConfig(siteID); err != nil {
		return fmt.Errorf("apply nginx config: %w", err)
	}

	return nil
}

// GetCertificateExpiry returns the expiration date of a certificate
func (s *SSLService) GetCertificateExpiry(domain string) (*time.Time, error) {
	certPath := filepath.Join("/etc/letsencrypt/live", domain, "fullchain.pem")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, ErrCertNotFound
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	return &cert.NotAfter, nil
}

// GetCertificateInfo returns detailed certificate information
func (s *SSLService) GetCertificateInfo(domain string) (*CertificateInfo, error) {
	certPath := filepath.Join("/etc/letsencrypt/live", domain, "fullchain.pem")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, ErrCertNotFound
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	return &CertificateInfo{
		Domain:          domain,
		Issuer:          cert.Issuer.CommonName,
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		DNSNames:        cert.DNSNames,
		IsExpired:       time.Now().After(cert.NotAfter),
		DaysUntilExpiry: int(time.Until(cert.NotAfter).Hours() / 24),
	}, nil
}

// CheckAndUpdateSSLStatus checks certificate status for a site and updates DB
func (s *SSLService) CheckAndUpdateSSLStatus(siteID int64) error {
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return err
	}

	expiresAt, err := s.GetCertificateExpiry(site.Name)
	if err != nil {
		// Certificate not found or invalid - mark as not SSL
		site.SSLEnabled = false
		site.SSLExpiresAt = nil
	} else {
		site.SSLEnabled = true
		site.SSLExpiresAt = expiresAt
	}

	return s.siteRepo.Update(site)
}

// RenewCertificates runs certbot renew for all certificates
func (s *SSLService) RenewCertificates() error {
	args := []string{"renew", "--non-interactive"}

	if s.config.SSL.Staging {
		args = append(args, "--staging")
	}

	cmd := exec.Command("sudo", append([]string{"certbot"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	// Check if any certificates were renewed
	if strings.Contains(string(output), "renewed") {
		// Reload nginx to pick up new certificates
		return s.nginxService.Reload()
	}

	return nil
}

// RevokeCertificate revokes a certificate for a site
func (s *SSLService) RevokeCertificate(siteID int64) error {
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return err
	}

	certPath := filepath.Join("/etc/letsencrypt/live", site.Name, "cert.pem")

	args := []string{
		"revoke",
		"--cert-path", certPath,
		"--non-interactive",
	}

	cmd := exec.Command("sudo", append([]string{"certbot"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	// Update site SSL status
	site.SSLEnabled = false
	site.SSLExpiresAt = nil
	if err := s.siteRepo.Update(site); err != nil {
		return fmt.Errorf("update site SSL status: %w", err)
	}

	// Regenerate nginx config without SSL
	if err := s.nginxService.ApplyConfig(siteID); err != nil {
		return fmt.Errorf("apply nginx config: %w", err)
	}

	return nil
}

// DeleteCertificate deletes a certificate for a site
func (s *SSLService) DeleteCertificate(siteID int64) error {
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return err
	}

	args := []string{
		"delete",
		"--cert-name", site.Name,
		"--non-interactive",
	}

	cmd := exec.Command("sudo", append([]string{"certbot"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	// Update site SSL status
	site.SSLEnabled = false
	site.SSLExpiresAt = nil
	return s.siteRepo.Update(site)
}

// CertificateInfo holds parsed certificate information
type CertificateInfo struct {
	Domain          string
	Issuer          string
	NotBefore       time.Time
	NotAfter        time.Time
	DNSNames        []string
	IsExpired       bool
	DaysUntilExpiry int
}

// IsCertificateValid checks if a certificate exists and is valid
func (s *SSLService) IsCertificateValid(domain string) bool {
	info, err := s.GetCertificateInfo(domain)
	if err != nil {
		return false
	}
	return !info.IsExpired && info.DaysUntilExpiry > 0
}
