package proprojects

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SharedProjectRow struct {
	ID                   uint      `gorm:"column:id"`
	Name                 string    `gorm:"column:name"`
	Description          string    `gorm:"column:description"`
	RootPath             string    `gorm:"column:root_path"`
	Token                string    `gorm:"column:token"`
	IsPaused             bool      `gorm:"column:is_paused"`
	IdentityVerification bool      `gorm:"column:identity_verification_enabled"`
	BearerAuthEnabled    bool      `gorm:"column:bearer_auth_enabled"`
	OAuthRedirectURI     string    `gorm:"column:oauth_redirect_uri"`
	Prompt               string    `gorm:"column:prompt"`
	PromptProfilesJSON   string    `gorm:"column:prompt_profiles_json"`
	LlamaCppModelPath    string    `gorm:"column:llama_cpp_model_path"`
	LlamaCppModelName    string    `gorm:"column:llama_cpp_model_name"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (SharedProjectRow) TableName() string {
	return "projects"
}

type SharedRAGCollectionRow struct {
	ID                 uint      `gorm:"column:id"`
	CollectionID       string    `gorm:"column:collection_id"`
	Name               string    `gorm:"column:name"`
	DataType           string    `gorm:"column:data_type"`
	SourcePath         string    `gorm:"column:source_path"`
	ServiceMode        string    `gorm:"column:service_mode"`
	VectorConnectionID string    `gorm:"column:vector_connection_id"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (SharedRAGCollectionRow) TableName() string {
	return "rag_collections"
}

type SharedProjectRAGCollectionRow struct {
	ProjectID       uint      `gorm:"column:project_id"`
	RAGCollectionID uint      `gorm:"column:rag_collection_id"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (SharedProjectRAGCollectionRow) TableName() string {
	return "project_rag_collections"
}

type ProjectShadow struct {
	ProjectID             uint      `gorm:"primaryKey" json:"project_id"`
	Name                  string    `gorm:"size:255;not null" json:"name"`
	Description           string    `gorm:"type:text" json:"description"`
	RootPath              string    `gorm:"size:1024" json:"root_path"`
	Token                 string    `gorm:"size:64;index;not null" json:"token"`
	IsPaused              bool      `gorm:"not null;default:false" json:"is_paused"`
	IdentityVerification  bool      `gorm:"not null;default:false" json:"identity_verification_enabled"`
	BearerAuthEnabled     bool      `gorm:"not null;default:false" json:"bearer_auth_enabled"`
	OAuthRedirectURI      string    `gorm:"type:text" json:"oauth_redirect_uri"`
	Prompt                string    `gorm:"type:text" json:"prompt"`
	PromptProfilesJSON    string    `gorm:"type:text" json:"-"`
	LlamaCppModelPath     string    `gorm:"size:2048" json:"llama_cpp_model_path"`
	LlamaCppModelName     string    `gorm:"size:255" json:"llama_cpp_model_name"`
	SharedCreatedAt       time.Time `json:"shared_created_at"`
	SharedUpdatedAt       time.Time `json:"shared_updated_at"`
	LastShadowSyncedAtUTC time.Time `gorm:"index" json:"last_shadow_synced_at_utc"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type RAGCollectionShadow struct {
	RAGCollectionID       uint      `gorm:"primaryKey" json:"rag_collection_id"`
	CollectionID          string    `gorm:"size:255;uniqueIndex;not null" json:"collection_id"`
	Name                  string    `gorm:"size:255;not null" json:"name"`
	DataType              string    `gorm:"size:64;not null" json:"data_type"`
	SourcePath            string    `gorm:"type:text" json:"source_path"`
	ServiceMode           string    `gorm:"size:64" json:"service_mode"`
	VectorConnectionID    string    `gorm:"size:255" json:"vector_connection_id"`
	SharedCreatedAt       time.Time `json:"shared_created_at"`
	SharedUpdatedAt       time.Time `json:"shared_updated_at"`
	LastShadowSyncedAtUTC time.Time `gorm:"index" json:"last_shadow_synced_at_utc"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type ProjectRAGCollectionShadow struct {
	ProjectID             uint      `gorm:"primaryKey" json:"project_id"`
	RAGCollectionID       uint      `gorm:"primaryKey" json:"rag_collection_id"`
	LastShadowSyncedAtUTC time.Time `gorm:"index" json:"last_shadow_synced_at_utc"`
	SharedCreatedAt       time.Time `json:"shared_created_at"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Service struct {
	stateDB  *gorm.DB
	sharedDB *gorm.DB
}

type ShadowStatus struct {
	ProjectsCount            int64      `json:"projects_count"`
	RAGCollectionsCount      int64      `json:"rag_collections_count"`
	ProjectRAGLinksCount     int64      `json:"project_rag_links_count"`
	LastProjectSyncAt        *time.Time `json:"last_project_sync_at,omitempty"`
	LastRAGCollectionSyncAt  *time.Time `json:"last_rag_collection_sync_at,omitempty"`
	LastProjectRAGLinkSyncAt *time.Time `json:"last_project_rag_link_sync_at,omitempty"`
}

func NewService(stateDB, sharedDB *gorm.DB) (*Service, error) {
	if stateDB == nil {
		return nil, errors.New("project shadow state database is not available")
	}
	if sharedDB == nil {
		return nil, errors.New("project shadow shared database is not available")
	}
	return &Service{
		stateDB:  stateDB,
		sharedDB: sharedDB,
	}, nil
}

func (s *Service) AutoMigrate(ctx context.Context) error {
	if s == nil || s.stateDB == nil {
		return errors.New("project shadow state database is not initialized")
	}
	return s.stateDB.WithContext(ctx).AutoMigrate(
		&ProjectShadow{},
		&RAGCollectionShadow{},
		&ProjectRAGCollectionShadow{},
	)
}

func (s *Service) RebindStateDB(ctx context.Context, db *gorm.DB) error {
	if s == nil {
		return errors.New("project shadow service is not initialized")
	}
	if db == nil {
		return errors.New("project shadow state database is not available")
	}
	s.stateDB = db
	return s.AutoMigrate(ctx)
}

func (s *Service) SyncAll(ctx context.Context) (int, error) {
	if s == nil || s.stateDB == nil || s.sharedDB == nil {
		return 0, errors.New("project shadow databases are not initialized")
	}

	var rows []SharedProjectRow
	if err := s.sharedDB.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	for _, row := range rows {
		shadow := ProjectShadow{
			ProjectID:             row.ID,
			Name:                  row.Name,
			Description:           row.Description,
			RootPath:              row.RootPath,
			Token:                 row.Token,
			IsPaused:              row.IsPaused,
			IdentityVerification:  row.IdentityVerification,
			BearerAuthEnabled:     row.BearerAuthEnabled,
			OAuthRedirectURI:      row.OAuthRedirectURI,
			Prompt:                row.Prompt,
			PromptProfilesJSON:    row.PromptProfilesJSON,
			LlamaCppModelPath:     row.LlamaCppModelPath,
			LlamaCppModelName:     row.LlamaCppModelName,
			SharedCreatedAt:       row.CreatedAt,
			SharedUpdatedAt:       row.UpdatedAt,
			LastShadowSyncedAtUTC: now,
		}
		if err := s.stateDB.WithContext(ctx).Save(&shadow).Error; err != nil {
			return 0, err
		}
	}

	var ragRows []SharedRAGCollectionRow
	if err := s.sharedDB.WithContext(ctx).Order("id asc").Find(&ragRows).Error; err != nil && !isSharedShadowOptionalTableError(err) {
		return 0, err
	}
	for _, row := range ragRows {
		shadow := RAGCollectionShadow{
			RAGCollectionID:       row.ID,
			CollectionID:          row.CollectionID,
			Name:                  row.Name,
			DataType:              row.DataType,
			SourcePath:            row.SourcePath,
			ServiceMode:           row.ServiceMode,
			VectorConnectionID:    row.VectorConnectionID,
			SharedCreatedAt:       row.CreatedAt,
			SharedUpdatedAt:       row.UpdatedAt,
			LastShadowSyncedAtUTC: now,
		}
		if err := s.stateDB.WithContext(ctx).Save(&shadow).Error; err != nil {
			return 0, err
		}
	}

	var linkRows []SharedProjectRAGCollectionRow
	if err := s.sharedDB.WithContext(ctx).Order("project_id asc, rag_collection_id asc").Find(&linkRows).Error; err != nil && !isSharedShadowOptionalTableError(err) {
		return 0, err
	}
	for _, row := range linkRows {
		shadow := ProjectRAGCollectionShadow{
			ProjectID:             row.ProjectID,
			RAGCollectionID:       row.RAGCollectionID,
			SharedCreatedAt:       row.CreatedAt,
			LastShadowSyncedAtUTC: now,
		}
		if err := s.stateDB.WithContext(ctx).Save(&shadow).Error; err != nil {
			return 0, err
		}
	}

	return len(rows), nil
}

func (s *Service) SyncProjectByID(ctx context.Context, projectID uint) error {
	if s == nil || s.stateDB == nil || s.sharedDB == nil {
		return errors.New("project shadow databases are not initialized")
	}

	var row SharedProjectRow
	if err := s.sharedDB.WithContext(ctx).First(&row, projectID).Error; err != nil {
		return err
	}

	now := time.Now().UTC()
	shadow := ProjectShadow{
		ProjectID:             row.ID,
		Name:                  row.Name,
		Description:           row.Description,
		RootPath:              row.RootPath,
		Token:                 row.Token,
		IsPaused:              row.IsPaused,
		IdentityVerification:  row.IdentityVerification,
		BearerAuthEnabled:     row.BearerAuthEnabled,
		OAuthRedirectURI:      row.OAuthRedirectURI,
		Prompt:                row.Prompt,
		PromptProfilesJSON:    row.PromptProfilesJSON,
		LlamaCppModelPath:     row.LlamaCppModelPath,
		LlamaCppModelName:     row.LlamaCppModelName,
		SharedCreatedAt:       row.CreatedAt,
		SharedUpdatedAt:       row.UpdatedAt,
		LastShadowSyncedAtUTC: now,
	}
	return s.stateDB.WithContext(ctx).Save(&shadow).Error
}

func (s *Service) SyncRAGCollectionByCollectionID(ctx context.Context, collectionID string) error {
	if s == nil || s.stateDB == nil || s.sharedDB == nil {
		return errors.New("project shadow databases are not initialized")
	}

	var row SharedRAGCollectionRow
	if err := s.sharedDB.WithContext(ctx).Where("collection_id = ?", strings.TrimSpace(collectionID)).First(&row).Error; err != nil {
		return err
	}
	return s.syncRAGCollectionRow(ctx, row)
}

func (s *Service) SyncProjectRAGLink(ctx context.Context, projectID uint, collectionID string) error {
	if s == nil || s.stateDB == nil || s.sharedDB == nil {
		return errors.New("project shadow databases are not initialized")
	}

	var collection SharedRAGCollectionRow
	if err := s.sharedDB.WithContext(ctx).Where("collection_id = ?", strings.TrimSpace(collectionID)).First(&collection).Error; err != nil {
		return err
	}
	if err := s.syncRAGCollectionRow(ctx, collection); err != nil {
		return err
	}

	var link SharedProjectRAGCollectionRow
	if err := s.sharedDB.WithContext(ctx).
		Where("project_id = ? AND rag_collection_id = ?", projectID, collection.ID).
		First(&link).Error; err != nil {
		return err
	}

	now := time.Now().UTC()
	shadow := ProjectRAGCollectionShadow{
		ProjectID:             link.ProjectID,
		RAGCollectionID:       link.RAGCollectionID,
		SharedCreatedAt:       link.CreatedAt,
		LastShadowSyncedAtUTC: now,
	}
	return s.stateDB.WithContext(ctx).Save(&shadow).Error
}

func (s *Service) GetByProjectID(ctx context.Context, projectID uint) (*ProjectShadow, error) {
	if s == nil || s.stateDB == nil {
		return nil, errors.New("project shadow state database is not initialized")
	}
	var shadow ProjectShadow
	err := s.stateDB.WithContext(ctx).Where("project_id = ?", projectID).First(&shadow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &shadow, err
}

func (s *Service) GetRAGCollectionByCollectionID(ctx context.Context, collectionID string) (*RAGCollectionShadow, error) {
	if s == nil || s.stateDB == nil {
		return nil, errors.New("project shadow state database is not initialized")
	}
	var shadow RAGCollectionShadow
	err := s.stateDB.WithContext(ctx).Where("collection_id = ?", strings.TrimSpace(collectionID)).First(&shadow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &shadow, err
}

func (s *Service) GetRAGCollectionByDBID(ctx context.Context, collectionDBID uint) (*RAGCollectionShadow, error) {
	if s == nil || s.stateDB == nil {
		return nil, errors.New("project shadow state database is not initialized")
	}
	var shadow RAGCollectionShadow
	err := s.stateDB.WithContext(ctx).Where("rag_collection_id = ?", collectionDBID).First(&shadow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &shadow, err
}

func (s *Service) HasProjectRAGCollectionLink(ctx context.Context, projectID, collectionDBID uint) (bool, error) {
	if s == nil || s.stateDB == nil {
		return false, errors.New("project shadow state database is not initialized")
	}
	var shadow ProjectRAGCollectionShadow
	err := s.stateDB.WithContext(ctx).
		Where("project_id = ? AND rag_collection_id = ?", projectID, collectionDBID).
		First(&shadow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) Status(ctx context.Context) (ShadowStatus, error) {
	if s == nil || s.stateDB == nil {
		return ShadowStatus{}, errors.New("project shadow state database is not initialized")
	}

	status := ShadowStatus{}

	if err := s.stateDB.WithContext(ctx).Model(&ProjectShadow{}).Count(&status.ProjectsCount).Error; err != nil {
		return ShadowStatus{}, err
	}
	if err := s.stateDB.WithContext(ctx).Model(&RAGCollectionShadow{}).Count(&status.RAGCollectionsCount).Error; err != nil {
		return ShadowStatus{}, err
	}
	if err := s.stateDB.WithContext(ctx).Model(&ProjectRAGCollectionShadow{}).Count(&status.ProjectRAGLinksCount).Error; err != nil {
		return ShadowStatus{}, err
	}

	var projectShadow ProjectShadow
	if err := s.stateDB.WithContext(ctx).Order("last_shadow_synced_at_utc desc").First(&projectShadow).Error; err == nil {
		value := projectShadow.LastShadowSyncedAtUTC
		status.LastProjectSyncAt = &value
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ShadowStatus{}, err
	}

	var ragShadow RAGCollectionShadow
	if err := s.stateDB.WithContext(ctx).Order("last_shadow_synced_at_utc desc").First(&ragShadow).Error; err == nil {
		value := ragShadow.LastShadowSyncedAtUTC
		status.LastRAGCollectionSyncAt = &value
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ShadowStatus{}, err
	}

	var linkShadow ProjectRAGCollectionShadow
	if err := s.stateDB.WithContext(ctx).Order("last_shadow_synced_at_utc desc").First(&linkShadow).Error; err == nil {
		value := linkShadow.LastShadowSyncedAtUTC
		status.LastProjectRAGLinkSyncAt = &value
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ShadowStatus{}, err
	}

	return status, nil
}

func (s *Service) syncRAGCollectionRow(ctx context.Context, row SharedRAGCollectionRow) error {
	now := time.Now().UTC()
	shadow := RAGCollectionShadow{
		RAGCollectionID:       row.ID,
		CollectionID:          row.CollectionID,
		Name:                  row.Name,
		DataType:              row.DataType,
		SourcePath:            row.SourcePath,
		ServiceMode:           row.ServiceMode,
		VectorConnectionID:    row.VectorConnectionID,
		SharedCreatedAt:       row.CreatedAt,
		SharedUpdatedAt:       row.UpdatedAt,
		LastShadowSyncedAtUTC: now,
	}
	return s.stateDB.WithContext(ctx).Save(&shadow).Error
}

func isSharedShadowOptionalTableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "no such table: rag_collections") ||
		strings.Contains(message, "no such table: project_rag_collections") ||
		strings.Contains(message, "relation \"rag_collections\" does not exist") ||
		strings.Contains(message, "relation \"project_rag_collections\" does not exist")
}
