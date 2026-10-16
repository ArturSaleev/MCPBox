package proprojects

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSyncAllCopiesSharedProjectsIntoShadowState(t *testing.T) {
	sharedDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shared.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(shared) error = %v", err)
	}
	stateDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "state.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(state) error = %v", err)
	}

	if err := sharedDB.AutoMigrate(&SharedProjectRow{}); err != nil {
		t.Fatalf("shared AutoMigrate() error = %v", err)
	}
	if err := sharedDB.Create(&SharedProjectRow{
		ID:                   7,
		Name:                 "Enterprise Project",
		Token:                "token-7",
		IsPaused:             true,
		IdentityVerification: true,
	}).Error; err != nil {
		t.Fatalf("shared Create() error = %v", err)
	}

	service, err := NewService(stateDB, sharedDB)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	count, err := service.SyncAll(context.Background())
	if err != nil {
		t.Fatalf("SyncAll() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("SyncAll() count = %d, want 1", count)
	}

	shadow, err := service.GetByProjectID(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetByProjectID() error = %v", err)
	}
	if shadow == nil {
		t.Fatal("GetByProjectID() returned nil shadow")
	}
	if shadow.Token != "token-7" {
		t.Fatalf("shadow.Token = %q, want %q", shadow.Token, "token-7")
	}
	if !shadow.IdentityVerification {
		t.Fatal("shadow.IdentityVerification = false, want true")
	}
}

func TestSyncAllCopiesRAGCollectionsAndLinksIntoShadowState(t *testing.T) {
	sharedDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shared-rag.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(shared) error = %v", err)
	}
	stateDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "state-rag.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(state) error = %v", err)
	}

	if err := sharedDB.AutoMigrate(&SharedProjectRow{}, &SharedRAGCollectionRow{}, &SharedProjectRAGCollectionRow{}); err != nil {
		t.Fatalf("shared AutoMigrate() error = %v", err)
	}
	if err := sharedDB.Create(&SharedProjectRow{ID: 3, Name: "P3", Token: "p3"}).Error; err != nil {
		t.Fatalf("Create project error = %v", err)
	}
	if err := sharedDB.Create(&SharedRAGCollectionRow{ID: 9, CollectionID: "kb-9", Name: "KB 9", ServiceMode: "ragbox_only"}).Error; err != nil {
		t.Fatalf("Create rag collection error = %v", err)
	}
	if err := sharedDB.Create(&SharedProjectRAGCollectionRow{ProjectID: 3, RAGCollectionID: 9}).Error; err != nil {
		t.Fatalf("Create project-rag link error = %v", err)
	}

	service, err := NewService(stateDB, sharedDB)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if _, err := service.SyncAll(context.Background()); err != nil {
		t.Fatalf("SyncAll() error = %v", err)
	}

	collection, err := service.GetRAGCollectionByCollectionID(context.Background(), "kb-9")
	if err != nil {
		t.Fatalf("GetRAGCollectionByCollectionID() error = %v", err)
	}
	if collection == nil || collection.RAGCollectionID != 9 {
		t.Fatalf("unexpected collection shadow = %#v", collection)
	}

	linked, err := service.HasProjectRAGCollectionLink(context.Background(), 3, 9)
	if err != nil {
		t.Fatalf("HasProjectRAGCollectionLink() error = %v", err)
	}
	if !linked {
		t.Fatal("HasProjectRAGCollectionLink() = false, want true")
	}

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.ProjectsCount != 1 {
		t.Fatalf("ProjectsCount = %d, want 1", status.ProjectsCount)
	}
	if status.RAGCollectionsCount != 1 {
		t.Fatalf("RAGCollectionsCount = %d, want 1", status.RAGCollectionsCount)
	}
	if status.ProjectRAGLinksCount != 1 {
		t.Fatalf("ProjectRAGLinksCount = %d, want 1", status.ProjectRAGLinksCount)
	}
}
