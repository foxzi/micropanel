package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCreateSiteRequest_Binding(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantName   string
		wantSSL    *bool
	}{
		{
			name:     "valid request with name only",
			body:     `{"name": "example.com"}`,
			wantErr:  false,
			wantName: "example.com",
			wantSSL:  nil,
		},
		{
			name:     "valid request with ssl true",
			body:     `{"name": "example.com", "ssl": true}`,
			wantErr:  false,
			wantName: "example.com",
			wantSSL:  ptrBool(true),
		},
		{
			name:     "valid request with ssl false",
			body:     `{"name": "example.com", "ssl": false}`,
			wantErr:  false,
			wantName: "example.com",
			wantSSL:  ptrBool(false),
		},
		{
			name:    "missing name",
			body:    `{"ssl": true}`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/sites", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req createSiteRequest
			err := c.ShouldBindJSON(&req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if req.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", req.Name, tt.wantName)
			}

			if tt.wantSSL == nil {
				if req.SSL != nil {
					t.Errorf("SSL = %v, want nil", *req.SSL)
				}
			} else {
				if req.SSL == nil {
					t.Errorf("SSL = nil, want %v", *tt.wantSSL)
				} else if *req.SSL != *tt.wantSSL {
					t.Errorf("SSL = %v, want %v", *req.SSL, *tt.wantSSL)
				}
			}
		})
	}
}

func TestSiteResponse_JSON(t *testing.T) {
	resp := siteResponse{
		ID:         1,
		Name:       "example.com",
		IsEnabled:  true,
		SSLEnabled: false,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded["id"].(float64) != 1 {
		t.Errorf("id = %v, want 1", decoded["id"])
	}
	if decoded["name"].(string) != "example.com" {
		t.Errorf("name = %v, want example.com", decoded["name"])
	}
	if decoded["is_enabled"].(bool) != true {
		t.Errorf("is_enabled = %v, want true", decoded["is_enabled"])
	}
	if decoded["ssl_enabled"].(bool) != false {
		t.Errorf("ssl_enabled = %v, want false", decoded["ssl_enabled"])
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	resp := errorResponse{Error: "something went wrong"}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	expected := `{"error":"something went wrong"}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", string(data), expected)
	}
}

func TestDeployResponse_JSON(t *testing.T) {
	resp := deployResponse{
		DeployID: 42,
		Status:   "success",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded["deploy_id"].(float64) != 42 {
		t.Errorf("deploy_id = %v, want 42", decoded["deploy_id"])
	}
	if decoded["status"].(string) != "success" {
		t.Errorf("status = %v, want success", decoded["status"])
	}
}

// Helper to create bool pointer
func ptrBool(b bool) *bool {
	return &b
}

func TestSSLDefaultBehavior(t *testing.T) {
	// Test that SSL defaults to true when not specified
	var req createSiteRequest
	json.Unmarshal([]byte(`{"name": "test.com"}`), &req)

	issueSSL := req.SSL == nil || *req.SSL
	if !issueSSL {
		t.Error("SSL should default to true when not specified")
	}

	// Test explicit false
	json.Unmarshal([]byte(`{"name": "test.com", "ssl": false}`), &req)
	issueSSL = req.SSL == nil || *req.SSL
	if issueSSL {
		t.Error("SSL should be false when explicitly set to false")
	}

	// Test explicit true
	json.Unmarshal([]byte(`{"name": "test.com", "ssl": true}`), &req)
	issueSSL = req.SSL == nil || *req.SSL
	if !issueSSL {
		t.Error("SSL should be true when explicitly set to true")
	}
}
