package models

import "testing"

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{
			name:     "admin role",
			role:     RoleAdmin,
			expected: true,
		},
		{
			name:     "user role",
			role:     RoleUser,
			expected: false,
		},
		{
			name:     "empty role",
			role:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Role: tt.role}
			if got := user.IsAdmin(); got != tt.expected {
				t.Errorf("IsAdmin() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestRole_Constants(t *testing.T) {
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin = %q, expected %q", RoleAdmin, "admin")
	}
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, expected %q", RoleUser, "user")
	}
}
