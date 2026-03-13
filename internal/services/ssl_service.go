package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"micropanel/internal/config"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

const certbotTimeout = 5 * time.Minute

// runCertbot executes certbot with timeout
func runCertbot(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), certbotTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sudo", append([]string{"certbot"}, args...)...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("%w: command timed out after %v", ErrCertbotFailed, certbotTimeout)
	}
	if err != nil {
		return output, fmt.Errorf("%w: %s", ErrCertbotFailed, string(output))
	}
	return output, nil
}

const certbotWebroot = "/var/www/certbot"

var (
	ErrCertbotFailed  = errors.New("certbot command failed")
	ErrCertbotBusy    = errors.New("another certbot operation is in progress")
	ErrNoDomains      = errors.New("no domains configured for site")
	ErrCertNotFound   = errors.New("certificate not found")
	ErrDomainNotFound = errors.New("domain not found")
)

type SSLService struct {
	config       *config.Config
	siteRepo     *repository.SiteRepository
	domainRepo   *repository.DomainRepository
	nginxService *NginxService
	certbotMu    sync.Mutex
}

func NewSSLService(cfg *config.Config, siteRepo *repository.SiteRepository, domainRepo *repository.DomainRepository, nginxService *NginxService) *SSLService {
	return &SSLService{
		config:       cfg,
		siteRepo:     siteRepo,
		domainRepo:   domainRepo,
		nginxService: nginxService,
	}
}

// IssueCertificate requests a new SSL certificate for site (primary domain + www + aliases).
// Uses certbot certonly --webroot to avoid conflicts with nginx config management.
// A mutex ensures only one certbot process runs at a time.
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

	// Serialize certbot calls to prevent "Another instance already running" errors
	s.certbotMu.Lock()
	defer s.certbotMu.Unlock()

	slog.Info("issuing SSL certificate", "site_id", siteID, "domain", site.Name, "hostnames", hostnames)

	// Run certbot with webroot plugin (doesn't modify nginx config)
	args := []string{
		"certonly",
		"--webroot",
		"-w", certbotWebroot,
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

	if _, err := runCertbot(args...); err != nil {
		slog.Error("certbot failed", "site_id", siteID, "domain", site.Name, "error", err)
		return err
	}

	// Update site SSL status and cert name
	expiresAt, _ := s.GetCertificateExpiry(site.Name)
	site.SSLEnabled = true
	site.SSLExpiresAt = expiresAt
	site.SSLCertName = site.Name
	if err := s.siteRepo.Update(site); err != nil {
		return fmt.Errorf("update site SSL status: %w", err)
	}

	// Regenerate nginx config with SSL block
	if err := s.nginxService.ApplyConfig(siteID); err != nil {
		return fmt.Errorf("apply nginx config: %w", err)
	}

	slog.Info("SSL certificate issued", "site_id", siteID, "domain", site.Name)
	return nil
}

// IssueCertificateForDomains requests an SSL certificate for specific domains only.
// Unlike IssueCertificate, this allows issuing certs for a subset of hostnames
// (e.g. only aliases when the primary domain DNS is not pointed to this server).
func (s *SSLService) IssueCertificateForDomains(siteID int64, domains []string) error {
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return fmt.Errorf("get site: %w", err)
	}

	if len(domains) == 0 {
		return ErrNoDomains
	}

	var domainArgs []string
	for _, d := range domains {
		domainArgs = append(domainArgs, "-d", d)
	}

	s.certbotMu.Lock()
	defer s.certbotMu.Unlock()

	// Use first domain as cert-name to avoid conflicts with primary domain cert
	certName := domains[0]

	slog.Info("issuing SSL certificate for specific domains", "site_id", siteID, "cert_name", certName, "domains", domains)

	args := []string{
		"certonly",
		"--webroot",
		"-w", certbotWebroot,
		"--email", s.config.SSL.Email,
		"--agree-tos",
		"--no-eff-email",
		"--non-interactive",
		"--cert-name", certName,
	}

	if s.config.SSL.Staging {
		args = append(args, "--staging")
	}

	args = append(args, domainArgs...)

	if _, err := runCertbot(args...); err != nil {
		slog.Error("certbot failed", "site_id", siteID, "domains", domains, "error", err)
		return err
	}

	// Update site SSL status and cert name
	expiresAt, _ := s.GetCertificateExpiry(certName)
	site.SSLEnabled = true
	site.SSLExpiresAt = expiresAt
	site.SSLCertName = certName
	if err := s.siteRepo.Update(site); err != nil {
		return fmt.Errorf("update site SSL status: %w", err)
	}

	// Regenerate nginx config with SSL block
	if err := s.nginxService.ApplyConfig(siteID); err != nil {
		return fmt.Errorf("apply nginx config: %w", err)
	}

	slog.Info("SSL certificate issued for specific domains", "site_id", siteID, "domains", domains)
	return nil
}

// GetCertificateExpiry returns the expiration date of a certificate
func (s *SSLService) GetCertificateExpiry(domain string) (*time.Time, error) {
	certPath := filepath.Join("/etc/letsencrypt/live", domain, "fullchain.pem")

	certPEM, err := s.readCertFile(certPath)
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

	certPEM, err := s.readCertFile(certPath)
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

// readCertFile reads a certificate file using sudo (micropanel user has no direct access)
func (s *SSLService) readCertFile(path string) ([]byte, error) {
	cmd := exec.Command("sudo", "/usr/bin/cat", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

// CheckAndUpdateSSLStatus checks certificate status for a site and updates DB
func (s *SSLService) CheckAndUpdateSSLStatus(siteID int64) error {
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return err
	}

	certName := site.GetSSLCertName()
	expiresAt, err := s.GetCertificateExpiry(certName)
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
	s.certbotMu.Lock()
	defer s.certbotMu.Unlock()

	args := []string{"renew", "--non-interactive"}

	if s.config.SSL.Staging {
		args = append(args, "--staging")
	}

	slog.Info("renewing SSL certificates")

	output, err := runCertbot(args...)
	if err != nil {
		slog.Error("certbot renew failed", "error", err)
		return err
	}

	// Check if any certificates were renewed
	if strings.Contains(string(output), "renewed") {
		slog.Info("certificates renewed, reloading nginx")
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

	s.certbotMu.Lock()
	defer s.certbotMu.Unlock()

	certPath := filepath.Join("/etc/letsencrypt/live", site.Name, "cert.pem")

	args := []string{
		"revoke",
		"--cert-path", certPath,
		"--non-interactive",
	}

	slog.Info("revoking SSL certificate", "site_id", siteID, "domain", site.Name)

	if _, err := runCertbot(args...); err != nil {
		slog.Error("certbot revoke failed", "site_id", siteID, "error", err)
		return err
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

	s.certbotMu.Lock()
	defer s.certbotMu.Unlock()

	args := []string{
		"delete",
		"--cert-name", site.Name,
		"--non-interactive",
	}

	slog.Info("deleting SSL certificate", "site_id", siteID, "domain", site.Name)

	if _, err := runCertbot(args...); err != nil {
		slog.Error("certbot delete failed", "site_id", siteID, "error", err)
		return err
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
