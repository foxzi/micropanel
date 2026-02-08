package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestIPWhitelist(t *testing.T) {
	tests := []struct {
		name       string
		allowedIPs []string
		clientIP   string
		wantStatus int
	}{
		{
			name:       "empty whitelist allows all",
			allowedIPs: []string{},
			clientIP:   "192.168.1.100",
			wantStatus: http.StatusOK,
		},
		{
			name:       "exact IP match",
			allowedIPs: []string{"192.168.1.100"},
			clientIP:   "192.168.1.100",
			wantStatus: http.StatusOK,
		},
		{
			name:       "IP not in whitelist",
			allowedIPs: []string{"192.168.1.100"},
			clientIP:   "192.168.1.101",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "CIDR match",
			allowedIPs: []string{"192.168.1.0/24"},
			clientIP:   "192.168.1.50",
			wantStatus: http.StatusOK,
		},
		{
			name:       "CIDR no match",
			allowedIPs: []string{"192.168.1.0/24"},
			clientIP:   "192.168.2.50",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "multiple IPs one matches",
			allowedIPs: []string{"10.0.0.1", "192.168.1.100", "172.16.0.1"},
			clientIP:   "192.168.1.100",
			wantStatus: http.StatusOK,
		},
		{
			name:       "mixed IPs and CIDRs",
			allowedIPs: []string{"10.0.0.1", "192.168.0.0/16"},
			clientIP:   "192.168.100.50",
			wantStatus: http.StatusOK,
		},
		{
			name:       "IPv6 localhost",
			allowedIPs: []string{"127.0.0.1", "::1"},
			clientIP:   "127.0.0.1",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(IPWhitelist(tt.allowedIPs))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestIPWhitelist_InvalidEntries(t *testing.T) {
	// Invalid entries should be ignored
	allowedIPs := []string{"not-an-ip", "192.168.1.100", "also-invalid"}

	router := gin.New()
	router.Use(IPWhitelist(allowedIPs))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid IP should pass even with invalid entries in whitelist, got status %d", w.Code)
	}
}
