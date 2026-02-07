package services

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"micropanel/internal/models"
	"micropanel/internal/repository"
)

const (
	SettingServerName  = "server_name"
	SettingServerNotes = "server_notes"
)

type SettingsService struct {
	repo       *repository.SettingsRepository
	externalIP string
	mu         sync.RWMutex
}

func NewSettingsService(repo *repository.SettingsRepository) *SettingsService {
	return &SettingsService{
		repo: repo,
	}
}

func (s *SettingsService) FetchExternalIP() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ifconfig.me/ip")
	if err != nil {
		log.Printf("Failed to fetch external IP: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read external IP response: %v", err)
		return
	}

	s.mu.Lock()
	s.externalIP = strings.TrimSpace(string(body))
	s.mu.Unlock()

	log.Printf("External IP: %s", s.externalIP)
}

func (s *SettingsService) GetExternalIP() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.externalIP
}

func (s *SettingsService) GetServerInfo() *models.ServerInfo {
	settings, err := s.repo.GetAll()
	if err != nil {
		log.Printf("Failed to get settings: %v", err)
		settings = make(map[string]string)
	}

	return &models.ServerInfo{
		ExternalIP:  s.GetExternalIP(),
		ServerName:  settings[SettingServerName],
		ServerNotes: settings[SettingServerNotes],
	}
}

func (s *SettingsService) UpdateServerName(name string) error {
	return s.repo.Set(SettingServerName, name)
}

func (s *SettingsService) UpdateServerNotes(notes string) error {
	return s.repo.Set(SettingServerNotes, notes)
}

func (s *SettingsService) GetServerName() string {
	val, _ := s.repo.Get(SettingServerName)
	return val
}

func (s *SettingsService) GetServerNotes() string {
	val, _ := s.repo.Get(SettingServerNotes)
	return val
}
