package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

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
	return s.db.WithContext(ctx).Create(server).Error
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
		Where("auto_start = ?", true).
		Order("id asc").
		Find(&servers).Error
	return servers, err
}

func newProjectToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}
