package proragbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/proprojects"
	"gorm.io/gorm"
)

type SharedProject struct {
	ID          uint   `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Description string `gorm:"column:description"`
	RootPath    string `gorm:"column:root_path"`
}

func (SharedProject) TableName() string {
	return "projects"
}

type SharedRAGCollection struct {
	ID                 uint   `gorm:"column:id"`
	CollectionID       string `gorm:"column:collection_id"`
	Name               string `gorm:"column:name"`
	DataType           string `gorm:"column:data_type"`
	SourcePath         string `gorm:"column:source_path"`
	ServiceMode        string `gorm:"column:service_mode"`
	VectorConnectionID string `gorm:"column:vector_connection_id"`
}

func (SharedRAGCollection) TableName() string {
	return "rag_collections"
}

type SharedProjectRAGCollection struct {
	ProjectID       uint `gorm:"column:project_id"`
	RAGCollectionID uint `gorm:"column:rag_collection_id"`
}

func (SharedProjectRAGCollection) TableName() string {
	return "project_rag_collections"
}

type ProjectWorkspaceBinding struct {
	ProjectID                         uint       `gorm:"primaryKey" json:"project_id"`
	RagBoxWorkspaceID                 string     `gorm:"size:255;index" json:"ragbox_workspace_id"`
	RagBoxWorkspaceVectorConnectionID string     `gorm:"size:255" json:"ragbox_workspace_vector_connection_id"`
	SyncStatus                        string     `gorm:"size:64;not null;default:'pending'" json:"sync_status"`
	LastError                         string     `gorm:"type:text" json:"last_error"`
	LastSyncedAt                      *time.Time `json:"last_synced_at"`
	CreatedAt                         time.Time  `json:"created_at"`
	UpdatedAt                         time.Time  `json:"updated_at"`
}

type ProjectCollectionBinding struct {
	ProjectID                         uint       `gorm:"primaryKey" json:"project_id"`
	RAGCollectionID                   uint       `gorm:"primaryKey" json:"rag_collection_id"`
	RagBoxWorkspaceID                 string     `gorm:"size:255;index" json:"ragbox_workspace_id"`
	RagBoxWorkspaceVectorConnectionID string     `gorm:"size:255" json:"ragbox_workspace_vector_connection_id"`
	RagBoxCollectionID                string     `gorm:"size:255;index" json:"ragbox_collection_id"`
	SyncStatus                        string     `gorm:"size:64;not null;default:'pending'" json:"sync_status"`
	LastJobID                         string     `gorm:"size:255" json:"last_job_id"`
	LastError                         string     `gorm:"type:text" json:"last_error"`
	LastSyncedAt                      *time.Time `json:"last_synced_at"`
	CreatedAt                         time.Time  `json:"created_at"`
	UpdatedAt                         time.Time  `json:"updated_at"`
}

type BindingService struct {
	stateDB    *gorm.DB
	sharedDB   *gorm.DB
	shadowRoot string
}

type GlobalWorkspaceBinding struct {
	ID                                uint       `gorm:"primaryKey" json:"id"`
	RagBoxWorkspaceID                 string     `gorm:"size:255;index" json:"ragbox_workspace_id"`
	RagBoxWorkspaceVectorConnectionID string     `gorm:"size:255" json:"ragbox_workspace_vector_connection_id"`
	SyncStatus                        string     `gorm:"size:64;not null;default:'pending'" json:"sync_status"`
	LastError                         string     `gorm:"type:text" json:"last_error"`
	LastSyncedAt                      *time.Time `json:"last_synced_at"`
	CreatedAt                         time.Time  `json:"created_at"`
	UpdatedAt                         time.Time  `json:"updated_at"`
}

type KnowledgeBaseBinding struct {
	RAGCollectionID                   uint       `gorm:"primaryKey" json:"rag_collection_id"`
	RagBoxWorkspaceID                 string     `gorm:"size:255;index" json:"ragbox_workspace_id"`
	RagBoxWorkspaceVectorConnectionID string     `gorm:"size:255" json:"ragbox_workspace_vector_connection_id"`
	RagBoxCollectionID                string     `gorm:"size:255;index" json:"ragbox_collection_id"`
	SyncStatus                        string     `gorm:"size:64;not null;default:'pending'" json:"sync_status"`
	LastJobID                         string     `gorm:"size:255" json:"last_job_id"`
	LastError                         string     `gorm:"type:text" json:"last_error"`
	LastSyncedAt                      *time.Time `json:"last_synced_at"`
	CreatedAt                         time.Time  `json:"created_at"`
	UpdatedAt                         time.Time  `json:"updated_at"`
}

type KnowledgeBaseIngestToken struct {
	RAGCollectionID uint      `gorm:"primaryKey" json:"rag_collection_id"`
	PublicID        string    `gorm:"size:64;uniqueIndex;not null" json:"public_id"`
	SecretToken     string    `gorm:"size:128;not null" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type KnowledgeBaseFilesystemSyncItem struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	RAGCollectionID uint      `gorm:"index;not null" json:"rag_collection_id"`
	RelativePath    string    `gorm:"size:2048;not null" json:"relative_path"`
	ExternalID      string    `gorm:"size:255;not null" json:"external_id"`
	ContentHash     string    `gorm:"size:128;not null" json:"content_hash"`
	SizeBytes       int64     `gorm:"not null;default:0" json:"size_bytes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type EnsureKnowledgeBaseResult struct {
	Collection SharedRAGCollection      `json:"collection"`
	Workspace  GlobalWorkspaceBinding   `json:"workspace"`
	Binding    KnowledgeBaseBinding     `json:"binding"`
	Token      KnowledgeBaseIngestToken `json:"token"`
}

type KnowledgeBaseIntegrationStatus struct {
	Collection                  SharedRAGCollection   `json:"collection"`
	Binding                     *KnowledgeBaseBinding `json:"binding,omitempty"`
	TokenPublicID               string                `json:"token_public_id,omitempty"`
	TokenSecret                 string                `json:"token_secret,omitempty"`
	TokenConfigured             bool                  `json:"token_configured"`
	IngestURLPath               string                `json:"ingest_url_path,omitempty"`
	ShadowSourcePath            string                `json:"shadow_source_path,omitempty"`
	UsesShadowStorage           bool                  `json:"uses_shadow_storage"`
	RequestedVectorConnectionID string                `json:"requested_vector_connection_id,omitempty"`
}

type EnsureProjectWorkspaceResult struct {
	Project   SharedProject           `json:"project"`
	Binding   ProjectWorkspaceBinding `json:"binding"`
	Workspace Workspace               `json:"workspace"`
}

type EnsureProjectCollectionResult struct {
	Project          SharedProject            `json:"project"`
	Collection       SharedRAGCollection      `json:"collection"`
	Binding          ProjectCollectionBinding `json:"binding"`
	RagBoxCollection Collection               `json:"ragbox_collection"`
}

func NewBindingService(stateDB, sharedDB *gorm.DB, shadowRoot string) (*BindingService, error) {
	if stateDB == nil {
		return nil, errors.New("state database is not available")
	}
	if sharedDB == nil {
		return nil, errors.New("shared database is not available")
	}
	root := strings.TrimSpace(shadowRoot)
	if root == "" {
		root = ".mcpboxpro/rag-shadow"
	}
	return &BindingService{
		stateDB:    stateDB,
		sharedDB:   sharedDB,
		shadowRoot: root,
	}, nil
}

func (s *BindingService) RebindStateDB(ctx context.Context, db *gorm.DB) error {
	if s == nil {
		return errors.New("binding service is not initialized")
	}
	if db == nil {
		return errors.New("state database is not available")
	}
	s.stateDB = db
	return s.AutoMigrate(ctx)
}

func (s *BindingService) StateDB() *gorm.DB {
	if s == nil {
		return nil
	}
	return s.stateDB
}

func (s *BindingService) AutoMigrate(ctx context.Context) error {
	if s == nil || s.stateDB == nil {
		return errors.New("binding database is not initialized")
	}
	return s.stateDB.WithContext(ctx).AutoMigrate(
		&ProjectWorkspaceBinding{},
		&ProjectCollectionBinding{},
		&GlobalWorkspaceBinding{},
		&KnowledgeBaseBinding{},
		&KnowledgeBaseIngestToken{},
		&KnowledgeBaseFilesystemSyncItem{},
	)
}

func (s *BindingService) EnsureGlobalWorkspace(ctx context.Context, client *Client, preferredVectorConnectionID string) (*GlobalWorkspaceBinding, *Workspace, error) {
	if client == nil {
		return nil, nil, errors.New("ragbox client is not configured")
	}

	if existing, err := s.GetGlobalWorkspaceBinding(ctx); err != nil {
		return nil, nil, err
	} else if existing != nil && strings.TrimSpace(existing.RagBoxWorkspaceID) != "" {
		workspace, workspaceErr := client.GetWorkspace(ctx, existing.RagBoxWorkspaceID)
		if workspaceErr != nil {
			return nil, nil, workspaceErr
		}
		if strings.TrimSpace(existing.RagBoxWorkspaceVectorConnectionID) != strings.TrimSpace(workspace.VectorConnectionID) {
			existing.RagBoxWorkspaceVectorConnectionID = strings.TrimSpace(workspace.VectorConnectionID)
			now := time.Now().UTC()
			existing.LastSyncedAt = &now
			if saveErr := s.stateDB.WithContext(ctx).Save(existing).Error; saveErr != nil {
				return nil, nil, saveErr
			}
		}
		return existing, workspace, nil
	}

	workspace, err := client.CreateWorkspace(ctx, CreateWorkspaceRequest{
		ExternalRef:        "mcpboxpro-global-knowledge",
		Name:               "MCPBox Knowledge",
		Slug:               "mcpboxpro-knowledge",
		VectorConnectionID: strings.TrimSpace(preferredVectorConnectionID),
		Description:        "Global MCPBox knowledge workspace",
		Metadata: map[string]any{
			"mcpbox_edition": "pro",
			"scope":          "global_knowledge_bases",
		},
	})
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	binding := &GlobalWorkspaceBinding{
		ID:                                1,
		RagBoxWorkspaceID:                 workspace.ID,
		RagBoxWorkspaceVectorConnectionID: strings.TrimSpace(workspace.VectorConnectionID),
		SyncStatus:                        "active",
		LastError:                         "",
		LastSyncedAt:                      &now,
	}
	if err := s.stateDB.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, nil, err
	}
	return binding, workspace, nil
}

func (s *BindingService) EnsureKnowledgeBase(ctx context.Context, client *Client, collectionID string, existingWorkspaceID string) (*EnsureKnowledgeBaseResult, error) {
	collection, err := s.GetSharedRAGCollectionByCollectionID(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if !knowledgeBaseUsesRagBox(collection.ServiceMode) {
		return nil, errors.New("knowledge base is configured without RagBox")
	}

	preferredVectorConnectionID := strings.TrimSpace(collection.VectorConnectionID)
	workspaceBinding := &GlobalWorkspaceBinding{}
	workspaceVectorConnectionID := ""
	workspaceID := ""
	if strings.TrimSpace(existingWorkspaceID) != "" {
		workspace, workspaceErr := client.GetWorkspace(ctx, strings.TrimSpace(existingWorkspaceID))
		if workspaceErr != nil {
			return nil, workspaceErr
		}
		workspaceID = workspace.ID
		workspaceVectorConnectionID = strings.TrimSpace(workspace.VectorConnectionID)
		workspaceBinding.RagBoxWorkspaceID = workspaceID
		workspaceBinding.RagBoxWorkspaceVectorConnectionID = workspaceVectorConnectionID
	} else {
		workspace, resolvedWorkspace, err := s.EnsureGlobalWorkspace(ctx, client, preferredVectorConnectionID)
		if err != nil {
			return nil, err
		}

		workspaceBinding = workspace
		workspaceID = workspaceBinding.RagBoxWorkspaceID
		workspaceVectorConnectionID = strings.TrimSpace(workspaceBinding.RagBoxWorkspaceVectorConnectionID)
		if resolvedWorkspace != nil &&
			preferredVectorConnectionID != "" &&
			strings.TrimSpace(resolvedWorkspace.VectorConnectionID) != "" &&
			strings.TrimSpace(resolvedWorkspace.VectorConnectionID) != preferredVectorConnectionID {
			dedicatedWorkspace, createErr := client.CreateWorkspace(ctx, CreateWorkspaceRequest{
				ExternalRef:        fmt.Sprintf("mcpboxpro-kb-%s", collection.CollectionID),
				Name:               strings.TrimSpace(collection.Name),
				Slug:               slugify(collection.Name, collection.CollectionID),
				VectorConnectionID: preferredVectorConnectionID,
				Description:        "Dedicated MCPBox knowledge workspace",
				Metadata: map[string]any{
					"mcpbox_edition":           "pro",
					"scope":                    "global_knowledge_base_dedicated",
					"mcpbox_rag_collection":    collection.CollectionID,
					"mcpbox_rag_collection_id": collection.ID,
				},
			})
			if createErr != nil {
				return nil, createErr
			}
			workspaceID = dedicatedWorkspace.ID
			workspaceVectorConnectionID = strings.TrimSpace(dedicatedWorkspace.VectorConnectionID)
			workspaceBinding = &GlobalWorkspaceBinding{
				RagBoxWorkspaceID:                 workspaceID,
				RagBoxWorkspaceVectorConnectionID: workspaceVectorConnectionID,
			}
		}
	}

	binding, err := s.GetKnowledgeBaseBinding(ctx, collection.ID)
	if err != nil {
		return nil, err
	}
	if binding == nil || strings.TrimSpace(binding.RagBoxCollectionID) == "" {
		request := CreateCollectionRequest{
			ExternalRef: fmt.Sprintf("mcpboxpro-kb-%s", collection.CollectionID),
			Name:        strings.TrimSpace(collection.Name),
			Slug:        slugify(collection.Name, collection.CollectionID),
			Description: strings.TrimSpace(collection.SourcePath),
			Metadata: map[string]any{
				"mcpbox_rag_collection":    collection.CollectionID,
				"mcpbox_rag_collection_id": collection.ID,
				"data_type":                collection.DataType,
				"scope":                    "global_knowledge_base",
			},
		}

		ragboxCollection, err := client.CreateCollection(ctx, workspaceID, request)
		if err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		binding = &KnowledgeBaseBinding{
			RAGCollectionID:                   collection.ID,
			RagBoxWorkspaceID:                 workspaceID,
			RagBoxWorkspaceVectorConnectionID: strings.TrimSpace(ragboxCollection.VectorConnectionID),
			RagBoxCollectionID:                ragboxCollection.ID,
			SyncStatus:                        "active",
			LastError:                         "",
			LastSyncedAt:                      &now,
		}
		if err := s.stateDB.WithContext(ctx).Save(binding).Error; err != nil {
			return nil, err
		}
	}

	token, err := s.EnsureKnowledgeBaseIngestToken(ctx, collection.ID)
	if err != nil {
		return nil, err
	}

	return &EnsureKnowledgeBaseResult{
		Collection: *collection,
		Workspace:  *workspaceBinding,
		Binding:    *binding,
		Token:      *token,
	}, nil
}

func (s *BindingService) EnsureKnowledgeBaseIngestToken(ctx context.Context, collectionDBID uint) (*KnowledgeBaseIngestToken, error) {
	token, err := s.GetKnowledgeBaseIngestToken(ctx, collectionDBID)
	if err != nil {
		return nil, err
	}
	if token != nil && strings.TrimSpace(token.PublicID) != "" && strings.TrimSpace(token.SecretToken) != "" {
		return token, nil
	}

	publicID, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	secret, err := randomHex(24)
	if err != nil {
		return nil, err
	}

	token = &KnowledgeBaseIngestToken{
		RAGCollectionID: collectionDBID,
		PublicID:        publicID,
		SecretToken:     secret,
	}
	if err := s.stateDB.WithContext(ctx).Save(token).Error; err != nil {
		return nil, err
	}
	return token, nil
}

func (s *BindingService) GetGlobalWorkspaceBinding(ctx context.Context) (*GlobalWorkspaceBinding, error) {
	var binding GlobalWorkspaceBinding
	err := s.stateDB.WithContext(ctx).First(&binding, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &binding, err
}

func (s *BindingService) GetKnowledgeBaseBinding(ctx context.Context, collectionDBID uint) (*KnowledgeBaseBinding, error) {
	var binding KnowledgeBaseBinding
	err := s.stateDB.WithContext(ctx).Where("rag_collection_id = ?", collectionDBID).First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &binding, err
}

func (s *BindingService) GetKnowledgeBaseBindingByCollectionID(ctx context.Context, collectionID string) (*KnowledgeBaseBinding, *SharedRAGCollection, error) {
	collection, err := s.GetSharedRAGCollectionByCollectionID(ctx, collectionID)
	if err != nil || collection == nil {
		return nil, collection, err
	}
	binding, err := s.GetKnowledgeBaseBinding(ctx, collection.ID)
	return binding, collection, err
}

func (s *BindingService) GetKnowledgeBaseIngestToken(ctx context.Context, collectionDBID uint) (*KnowledgeBaseIngestToken, error) {
	var token KnowledgeBaseIngestToken
	err := s.stateDB.WithContext(ctx).Where("rag_collection_id = ?", collectionDBID).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (s *BindingService) GetKnowledgeBaseIngestTokenByPublicID(ctx context.Context, publicID string) (*KnowledgeBaseIngestToken, error) {
	var token KnowledgeBaseIngestToken
	err := s.stateDB.WithContext(ctx).Where("public_id = ?", strings.TrimSpace(publicID)).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (s *BindingService) ResolveShadowSourcePath(collection SharedRAGCollection) string {
	return filepath.Join(s.shadowRoot, collection.CollectionID, "source")
}

func (s *BindingService) EnsureShadowSourcePath(collection SharedRAGCollection) (string, error) {
	sourcePath := s.ResolveShadowSourcePath(collection)
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		return "", err
	}
	return sourcePath, nil
}

func (s *BindingService) ListKnowledgeBaseIntegrationStatuses(ctx context.Context) ([]KnowledgeBaseIntegrationStatus, error) {
	var collections []SharedRAGCollection
	if err := s.sharedDB.WithContext(ctx).Order("name asc").Find(&collections).Error; err != nil {
		return nil, err
	}

	globalWorkspaceBinding, globalWorkspaceErr := s.GetGlobalWorkspaceBinding(ctx)
	if globalWorkspaceErr != nil {
		return nil, globalWorkspaceErr
	}

	items := make([]KnowledgeBaseIntegrationStatus, 0, len(collections))
	for _, collection := range collections {
		var binding *KnowledgeBaseBinding
		var token *KnowledgeBaseIngestToken
		var err error

		if knowledgeBaseUsesRagBox(collection.ServiceMode) {
			binding, err = s.GetKnowledgeBaseBinding(ctx, collection.ID)
			if err != nil {
				return nil, err
			}
			if binding != nil &&
				strings.TrimSpace(binding.RagBoxWorkspaceVectorConnectionID) == "" &&
				globalWorkspaceBinding != nil &&
				strings.TrimSpace(globalWorkspaceBinding.RagBoxWorkspaceID) != "" &&
				strings.TrimSpace(binding.RagBoxWorkspaceID) == strings.TrimSpace(globalWorkspaceBinding.RagBoxWorkspaceID) {
				binding.RagBoxWorkspaceVectorConnectionID = strings.TrimSpace(globalWorkspaceBinding.RagBoxWorkspaceVectorConnectionID)
			}
			token, err = s.GetKnowledgeBaseIngestToken(ctx, collection.ID)
			if err != nil {
				return nil, err
			}
		}
		shadowPath := s.ResolveShadowSourcePath(collection)
		items = append(items, KnowledgeBaseIntegrationStatus{
			Collection:                  collection,
			Binding:                     binding,
			TokenPublicID:               tokenPublicID(token),
			TokenSecret:                 tokenSecret(token),
			TokenConfigured:             token != nil && strings.TrimSpace(token.SecretToken) != "",
			IngestURLPath:               ingestURLPathForToken(token),
			ShadowSourcePath:            shadowPath,
			UsesShadowStorage:           strings.TrimSpace(collection.SourcePath) == "",
			RequestedVectorConnectionID: strings.TrimSpace(collection.VectorConnectionID),
		})
	}
	return items, nil
}

func (s *BindingService) EnsureProjectWorkspace(ctx context.Context, client *Client, projectID uint, preferredVectorConnectionID string, existingWorkspaceID string) (*EnsureProjectWorkspaceResult, error) {
	project, err := s.GetSharedProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, gorm.ErrRecordNotFound
	}

	if existing, err := s.GetProjectWorkspaceBinding(ctx, projectID); err != nil {
		return nil, err
	} else if existing != nil && strings.TrimSpace(existing.RagBoxWorkspaceID) != "" {
		workspace, workspaceErr := client.GetWorkspace(ctx, existing.RagBoxWorkspaceID)
		if workspaceErr != nil {
			return nil, workspaceErr
		}
		if strings.TrimSpace(existing.RagBoxWorkspaceVectorConnectionID) != strings.TrimSpace(workspace.VectorConnectionID) {
			existing.RagBoxWorkspaceVectorConnectionID = strings.TrimSpace(workspace.VectorConnectionID)
			now := time.Now().UTC()
			existing.LastSyncedAt = &now
			if saveErr := s.stateDB.WithContext(ctx).Save(existing).Error; saveErr != nil {
				return nil, saveErr
			}
		}
		return &EnsureProjectWorkspaceResult{
			Project:   *project,
			Binding:   *existing,
			Workspace: *workspace,
		}, nil
	}

	if strings.TrimSpace(existingWorkspaceID) != "" {
		workspace, workspaceErr := client.GetWorkspace(ctx, strings.TrimSpace(existingWorkspaceID))
		if workspaceErr != nil {
			return nil, workspaceErr
		}

		now := time.Now().UTC()
		binding := &ProjectWorkspaceBinding{
			ProjectID:                         project.ID,
			RagBoxWorkspaceID:                 workspace.ID,
			RagBoxWorkspaceVectorConnectionID: strings.TrimSpace(workspace.VectorConnectionID),
			SyncStatus:                        "active",
			LastError:                         "",
			LastSyncedAt:                      &now,
		}
		if err := s.stateDB.WithContext(ctx).Save(binding).Error; err != nil {
			return nil, err
		}

		return &EnsureProjectWorkspaceResult{
			Project:   *project,
			Binding:   *binding,
			Workspace: *workspace,
		}, nil
	}

	request := CreateWorkspaceRequest{
		TenantID:           client.tenantID,
		ExternalRef:        fmt.Sprintf("mcpbox-project-%d", project.ID),
		Name:               strings.TrimSpace(project.Name),
		Slug:               slugify(project.Name, fmt.Sprintf("project-%d", project.ID)),
		VectorConnectionID: strings.TrimSpace(preferredVectorConnectionID),
		Description:        strings.TrimSpace(project.Description),
		Metadata: map[string]any{
			"mcpbox_project_id": project.ID,
			"root_path":         project.RootPath,
		},
	}

	workspace, err := client.CreateWorkspace(ctx, request)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	binding := &ProjectWorkspaceBinding{
		ProjectID:                         project.ID,
		RagBoxWorkspaceID:                 workspace.ID,
		RagBoxWorkspaceVectorConnectionID: strings.TrimSpace(workspace.VectorConnectionID),
		SyncStatus:                        "active",
		LastError:                         "",
		LastSyncedAt:                      &now,
	}
	if err := s.stateDB.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, err
	}

	return &EnsureProjectWorkspaceResult{
		Project:   *project,
		Binding:   *binding,
		Workspace: *workspace,
	}, nil
}

func (s *BindingService) EnsureProjectCollection(ctx context.Context, client *Client, projectID uint, collectionID string) (*EnsureProjectCollectionResult, error) {
	project, err := s.GetSharedProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, gorm.ErrRecordNotFound
	}

	collection, err := s.GetSharedRAGCollectionByCollectionID(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, gorm.ErrRecordNotFound
	}

	linked, err := s.ProjectHasCollectionLink(ctx, projectID, collection.ID)
	if err != nil {
		return nil, err
	}
	if !linked {
		return nil, errors.New("knowledge base is not linked to the selected project")
	}

	workspaceBinding, err := s.GetProjectWorkspaceBinding(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if workspaceBinding == nil || strings.TrimSpace(workspaceBinding.RagBoxWorkspaceID) == "" {
		workspaceResult, ensureErr := s.EnsureProjectWorkspace(ctx, client, projectID, strings.TrimSpace(collection.VectorConnectionID), "")
		if ensureErr != nil {
			return nil, ensureErr
		}
		workspaceBinding = &workspaceResult.Binding
	}

	request := CreateCollectionRequest{
		ExternalRef: fmt.Sprintf("mcpbox-project-%d-kb-%s", project.ID, collection.CollectionID),
		Name:        strings.TrimSpace(collection.Name),
		Slug:        slugify(collection.Name, collection.CollectionID),
		Description: strings.TrimSpace(collection.SourcePath),
		Metadata: map[string]any{
			"mcpbox_project_id":        project.ID,
			"mcpbox_rag_collection":    collection.CollectionID,
			"mcpbox_rag_collection_id": collection.ID,
			"data_type":                collection.DataType,
		},
	}

	ragboxCollection, err := client.CreateCollection(ctx, workspaceBinding.RagBoxWorkspaceID, request)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	binding := &ProjectCollectionBinding{
		ProjectID:                         project.ID,
		RAGCollectionID:                   collection.ID,
		RagBoxWorkspaceID:                 workspaceBinding.RagBoxWorkspaceID,
		RagBoxWorkspaceVectorConnectionID: strings.TrimSpace(ragboxCollection.VectorConnectionID),
		RagBoxCollectionID:                ragboxCollection.ID,
		SyncStatus:                        "active",
		LastError:                         "",
		LastSyncedAt:                      &now,
	}
	if err := s.stateDB.WithContext(ctx).Save(binding).Error; err != nil {
		return nil, err
	}

	return &EnsureProjectCollectionResult{
		Project:          *project,
		Collection:       *collection,
		Binding:          *binding,
		RagBoxCollection: *ragboxCollection,
	}, nil
}

func (s *BindingService) GetSharedProject(ctx context.Context, projectID uint) (*SharedProject, error) {
	var project SharedProject
	err := s.sharedDB.WithContext(ctx).First(&project, projectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &project, err
}

func (s *BindingService) GetSharedRAGCollectionByCollectionID(ctx context.Context, collectionID string) (*SharedRAGCollection, error) {
	var shadow proprojects.RAGCollectionShadow
	err := s.stateDB.WithContext(ctx).Where("collection_id = ?", strings.TrimSpace(collectionID)).First(&shadow).Error
	if err == nil {
		return &SharedRAGCollection{
			ID:                 shadow.RAGCollectionID,
			CollectionID:       shadow.CollectionID,
			Name:               shadow.Name,
			DataType:           shadow.DataType,
			SourcePath:         shadow.SourcePath,
			ServiceMode:        shadow.ServiceMode,
			VectorConnectionID: shadow.VectorConnectionID,
		}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !isShadowLookupFallbackError(err) {
		return nil, err
	}

	var collection SharedRAGCollection
	err = s.sharedDB.WithContext(ctx).Where("collection_id = ?", strings.TrimSpace(collectionID)).First(&collection).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &collection, err
}

func (s *BindingService) GetSharedRAGCollectionByDBID(ctx context.Context, collectionDBID uint) (*SharedRAGCollection, error) {
	var shadow proprojects.RAGCollectionShadow
	err := s.stateDB.WithContext(ctx).Where("rag_collection_id = ?", collectionDBID).First(&shadow).Error
	if err == nil {
		return &SharedRAGCollection{
			ID:                 shadow.RAGCollectionID,
			CollectionID:       shadow.CollectionID,
			Name:               shadow.Name,
			DataType:           shadow.DataType,
			SourcePath:         shadow.SourcePath,
			ServiceMode:        shadow.ServiceMode,
			VectorConnectionID: shadow.VectorConnectionID,
		}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !isShadowLookupFallbackError(err) {
		return nil, err
	}

	var collection SharedRAGCollection
	err = s.sharedDB.WithContext(ctx).First(&collection, collectionDBID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &collection, err
}

func (s *BindingService) ProjectHasCollectionLink(ctx context.Context, projectID, collectionDBID uint) (bool, error) {
	var shadow proprojects.ProjectRAGCollectionShadow
	err := s.stateDB.WithContext(ctx).
		Where("project_id = ? AND rag_collection_id = ?", projectID, collectionDBID).
		First(&shadow).Error
	if err == nil {
		return true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !isShadowLookupFallbackError(err) {
		return false, err
	}

	var link SharedProjectRAGCollection
	err = s.sharedDB.WithContext(ctx).
		Where("project_id = ? AND rag_collection_id = ?", projectID, collectionDBID).
		First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *BindingService) GetProjectWorkspaceBinding(ctx context.Context, projectID uint) (*ProjectWorkspaceBinding, error) {
	var binding ProjectWorkspaceBinding
	err := s.stateDB.WithContext(ctx).Where("project_id = ?", projectID).First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &binding, err
}

func (s *BindingService) GetProjectCollectionBinding(ctx context.Context, projectID, collectionDBID uint) (*ProjectCollectionBinding, error) {
	var binding ProjectCollectionBinding
	err := s.stateDB.WithContext(ctx).
		Where("project_id = ? AND rag_collection_id = ?", projectID, collectionDBID).
		First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &binding, err
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value, fallback string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = slugPattern.ReplaceAllString(raw, "-")
	raw = strings.Trim(raw, "-")
	if raw != "" {
		return raw
	}
	fallback = strings.ToLower(strings.TrimSpace(fallback))
	fallback = slugPattern.ReplaceAllString(fallback, "-")
	fallback = strings.Trim(fallback, "-")
	if fallback == "" {
		return "item"
	}
	return fallback
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func tokenPublicID(token *KnowledgeBaseIngestToken) string {
	if token == nil {
		return ""
	}
	return strings.TrimSpace(token.PublicID)
}

func ingestURLPathForToken(token *KnowledgeBaseIngestToken) string {
	publicID := tokenPublicID(token)
	if publicID == "" {
		return ""
	}
	return "/api/pro/ragbox/ingest/" + publicID
}

func tokenSecret(token *KnowledgeBaseIngestToken) string {
	if token == nil {
		return ""
	}
	return strings.TrimSpace(token.SecretToken)
}

func isShadowLookupFallbackError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "no such table: rag_collection_shadows") ||
		strings.Contains(message, "no such table: project_rag_collection_shadows") ||
		strings.Contains(message, "relation \"rag_collection_shadows\" does not exist") ||
		strings.Contains(message, "relation \"project_rag_collection_shadows\" does not exist")
}

func normalizeKnowledgeBaseServiceMode(value string) string {
	switch strings.TrimSpace(value) {
	case "bleve_only", "ragbox_only", "dual":
		return strings.TrimSpace(value)
	default:
		return "bleve_only"
	}
}

func knowledgeBaseUsesRagBox(mode string) bool {
	switch normalizeKnowledgeBaseServiceMode(mode) {
	case "ragbox_only", "dual":
		return true
	default:
		return false
	}
}

func knowledgeBaseUsesBleve(mode string) bool {
	switch normalizeKnowledgeBaseServiceMode(mode) {
	case "bleve_only", "dual":
		return true
	default:
		return false
	}
}
