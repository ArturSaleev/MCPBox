package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"MCPBox/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(dsn string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Project{}, &models.MCPServer{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) CreateProject(ctx context.Context, project *models.Project) error {
	if project.Token == "" {
		token, err := newProjectToken()
		if err != nil {
			return err
		}

		project.Token = token
	}

	return s.db.WithContext(ctx).Create(project).Error
}

func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.WithContext(ctx).
		Preload("Servers").
		Order("id asc").
		Find(&projects).Error
	return projects, err
}

func (s *Store) GetProject(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).
		Preload("Servers").
		First(&project, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &project, err
}

func (s *Store) GetProjectByToken(ctx context.Context, token string) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).
		Preload("Servers").
		Where("token = ?", token).
		First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &project, err
}

func (s *Store) AddServer(ctx context.Context, server *models.MCPServer) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(server).Error; err != nil {
			return err
		}

		return tx.Model(&models.Project{}).
			Where("id = ? AND primary_server_id IS NULL", server.ProjectID).
			Update("primary_server_id", server.ID).Error
	})
}

func (s *Store) GetServer(ctx context.Context, id uint) (*models.MCPServer, error) {
	var server models.MCPServer
	err := s.db.WithContext(ctx).First(&server, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &server, err
}

func (s *Store) ListAutoStartServers(ctx context.Context) ([]models.MCPServer, error) {
	var servers []models.MCPServer
	err := s.db.WithContext(ctx).
		Joins("JOIN projects ON projects.id = mcp_servers.project_id").
		Where("auto_start = ?", true).
		Where("mcp_servers.is_enabled = ?", true).
		Where("projects.is_paused = ?", false).
		Order("mcp_servers.id asc").
		Find(&servers).Error
	return servers, err
}

func (s *Store) SetPrimaryServer(ctx context.Context, projectID, serverID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project models.Project
		if err := tx.First(&project, projectID).Error; err != nil {
			return err
		}

		var server models.MCPServer
		err := tx.Where("id = ? AND project_id = ?", serverID, projectID).First(&server).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("server %d does not belong to project %d", serverID, projectID)
		}
		if err != nil {
			return err
		}

		return tx.Model(&models.Project{}).
			Where("id = ?", projectID).
			Update("primary_server_id", serverID).Error
	})
}

func newProjectToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func (s *Store) SetProjectPaused(ctx context.Context, projectID uint, paused bool) error {
	return s.db.WithContext(ctx).Model(&models.Project{}).
		Where("id = ?", projectID).
		Update("is_paused", paused).Error
}

func (s *Store) SetServerEnabled(ctx context.Context, serverID uint, enabled bool) error {
	return s.db.WithContext(ctx).Model(&models.MCPServer{}).
		Where("id = ?", serverID).
		Update("is_enabled", enabled).Error
}

func (s *Store) CreateAuditLog(ctx context.Context, entry *models.AuditLog) error {
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *Store) ListAuditLogs(ctx context.Context, projectID *uint, limit int) ([]models.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var logs []models.AuditLog
	query := s.db.WithContext(ctx).Order("id desc").Limit(limit)
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}

	err := query.Find(&logs).Error
	return logs, err
}
