package proauth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCreateOrUpdateLocalPasswordUserAllowsLogin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	service, err := NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	record, created, err := service.CreateOrUpdateLocalPasswordUser(context.Background(), CreateLocalUserInput{
		Email:       "admin@admin.com",
		DisplayName: "System Admin",
		Password:    "password123",
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateLocalPasswordUser() error = %v", err)
	}
	if !created {
		t.Fatalf("created = false, want true")
	}
	if record.AuthProvider != "local" {
		t.Fatalf("auth provider = %q, want %q", record.AuthProvider, "local")
	}

	hasLocalUsers, err := service.HasLocalPasswordUsers(context.Background())
	if err != nil {
		t.Fatalf("HasLocalPasswordUsers() error = %v", err)
	}
	if !hasLocalUsers {
		t.Fatal("expected local password users to exist")
	}

	loggedInUser, session, token, err := service.LoginLocal(context.Background(), LocalLoginInput{
		Email:         "admin@admin.com",
		Password:      "password123",
		ExpiresInDays: 30,
	})
	if err != nil {
		t.Fatalf("LoginLocal() error = %v", err)
	}
	if loggedInUser.Email != "admin@admin.com" {
		t.Fatalf("login email = %q", loggedInUser.Email)
	}
	if len(loggedInUser.Roles) != 1 || loggedInUser.Roles[0] != "admin" {
		t.Fatalf("login roles = %#v", loggedInUser.Roles)
	}
	if session == nil || session.ID == 0 {
		t.Fatalf("session = %#v", session)
	}
	if token == "" {
		t.Fatal("expected session token")
	}
}

func TestUpdateLocalPasswordAllowsLoginWithNewPassword(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	service, err := NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	record, err := service.CreateLocalUser(context.Background(), CreateLocalUserInput{
		Email:       "admin@admin.com",
		DisplayName: "System Admin",
		Password:    "password123",
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	})
	if err != nil {
		t.Fatalf("CreateLocalUser() error = %v", err)
	}

	if _, err := service.UpdateLocalPassword(context.Background(), UpdateLocalPasswordInput{
		UserID:   record.ID,
		Password: "newpassword456",
	}); err != nil {
		t.Fatalf("UpdateLocalPassword() error = %v", err)
	}

	if _, _, _, err := service.LoginLocal(context.Background(), LocalLoginInput{
		Email:         "admin@admin.com",
		Password:      "password123",
		ExpiresInDays: 30,
	}); err == nil {
		t.Fatal("expected old password login to fail")
	}

	loggedInUser, session, token, err := service.LoginLocal(context.Background(), LocalLoginInput{
		Email:         "admin@admin.com",
		Password:      "newpassword456",
		ExpiresInDays: 30,
	})
	if err != nil {
		t.Fatalf("LoginLocal() with new password error = %v", err)
	}
	if loggedInUser.ID != record.ID {
		t.Fatalf("login user id = %d, want %d", loggedInUser.ID, record.ID)
	}
	if session == nil || session.ID == 0 {
		t.Fatalf("session = %#v", session)
	}
	if token == "" {
		t.Fatal("expected session token")
	}
}
