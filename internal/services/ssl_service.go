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
	"micropanel/internal/repository"
)

var (
	ErrCertbotFailed   = errors.New("certbot command failed")
	ErrNoDomains       = errors.New("no domains configured for site")
	ErrCertNotFound    = errors.New("certificate not found")
	ErrDomainNotFound  = errors.New("domain not found")
)

type SSLService struct {
	config       *config.Config
	domainRepo   *repository.DomainRepository
	nginxService *NginxService
}

func NewSSLService(cfg *config.Config, domainRepo *repository.DomainRepository, nginxService *NginxService) *SSLService {
	return &SSLService{
		config:       cfg,
		domainRepo:   domainRepo,
		nginxService: nginxService,
	}
}

// IssueCertificate requests a new SSL certificate for all domains of a site
func (s *SSLService) IssueCertificate(siteID int64) error {
	domains, err := s.domainRepo.ListBySite(siteID)
	if err != nil {
		return fmt.Errorf("list domains: %w", err)
	}

	if len(domains) == 0 {
		return ErrNoDomains
	}

	// Build domain list for certbot
	var domainArgs []string
	var primaryDomain string
	for _, d := range domains {
		domainArgs = append(domainArgs, "-d", d.Hostname)
		if d.IsPrimary || primaryDomain == "" {
			primaryDomain = d.Hostname
		}
	}

	// Run certbot
	args := []string{
		"certonly",
		"--webroot",
		"--webroot-path", "/var/www/certbot",
		"--email", s.config.SSL.Email,
		"--agree-tos",
		"--no-eff-email",
		"--non-interactive",
		"--cert-name", primaryDomain,
	}

	if s.config.SSL.Staging {
		args = append(args, "--staging")
	}

	args = append(args, domainArgs...)

	cmd := exec.Command("certbot", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	// Update domain SSL status
	expiresAt, _ := s.GetCertificateExpiry(primaryDomain)
	for _, d := range domains {
		d.SSLEnabled = true
		d.SSLExpiresAt = expiresAt
		if err := s.domainRepo.Update(d); err != nil {
			return fmt.Errorf("update domain SSL status: %w", err)
		}
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
		Domain:    domain,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		DNSNames:  cert.DNSNames,
		IsExpired: time.Now().After(cert.NotAfter),
		DaysUntilExpiry: int(time.Until(cert.NotAfter).Hours() / 24),
	}, nil
}

// CheckAndUpdateSSLStatus checks certificate status for a domain and updates DB
func (s *SSLService) CheckAndUpdateSSLStatus(domainID int64) error {
	domain, err := s.domainRepo.GetByID(domainID)
	if err != nil {
		return err
	}

	expiresAt, err := s.GetCertificateExpiry(domain.Hostname)
	if err != nil {
		// Certificate not found or invalid - mark as not SSL
		domain.SSLEnabled = false
		domain.SSLExpiresAt = nil
	} else {
		domain.SSLEnabled = true
		domain.SSLExpiresAt = expiresAt
	}

	return s.domainRepo.Update(domain)
}

// RenewCertificates runs certbot renew for all certificates
func (s *SSLService) RenewCertificates() error {
	args := []string{"renew", "--non-interactive"}

	if s.config.SSL.Staging {
		args = append(args, "--staging")
	}

	cmd := exec.Command("certbot", args...)
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

// RevokeCertificate revokes a certificate for a domain
func (s *SSLService) RevokeCertificate(domain string) error {
	certPath := filepath.Join("/etc/letsencrypt/live", domain, "cert.pem")

	args := []string{
		"revoke",
		"--cert-path", certPath,
		"--non-interactive",
	}

	cmd := exec.Command("certbot", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	return nil
}

// DeleteCertificate deletes a certificate for a domain
func (s *SSLService) DeleteCertificate(domain string) error {
	args := []string{
		"delete",
		"--cert-name", domain,
		"--non-interactive",
	}

	cmd := exec.Command("certbot", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}

	return nil
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
