package prohttp

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestKnowledgeBaseZipUploadDoesNotRequireRagBoxConfigForBleveOnly(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	db, err := gorm.Open(sqlite.Open(filepath.Join(tempDir, "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&testProject{}, &testRAGCollection{}, &testProjectRAGCollection{}); err != nil {
		t.Fatalf("db.AutoMigrate() error = %v", err)
	}

	reindexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("reindex method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/rag/collections/kb-1/reindex" {
			t.Fatalf("reindex path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer reindexServer.Close()

	reindexHost, reindexPort := mustSplitHostPort(t, reindexServer.URL)
	runtime := &app.RuntimeContext{
		DB: db,
		Edition: app.Edition{
			AdminHost: reindexHost,
			AdminPort: reindexPort,
		},
	}

	authService, err := proauth.NewServiceFromRuntime(runtime)
	if err != nil {
		t.Fatalf("NewServiceFromRuntime(auth) error = %v", err)
	}
	if err := authService.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auth AutoMigrate() error = %v", err)
	}
	if _, _, err := authService.CreateOrUpdateLocalPasswordUser(context.Background(), proauth.CreateLocalUserInput{
		Email:       "admin@example.com",
		DisplayName: "System Admin",
		Password:    "password123",
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	}); err != nil {
		t.Fatalf("CreateOrUpdateLocalPasswordUser() error = %v", err)
	}

	ragboxService, err := proragbox.NewServiceFromRuntime(runtime)
	if err != nil {
		t.Fatalf("NewServiceFromRuntime(ragbox) error = %v", err)
	}
	if err := ragboxService.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("ragbox AutoMigrate() error = %v", err)
	}

	bindings, err := proragbox.NewBindingService(db, db, filepath.Join(tempDir, "rag-shadow"))
	if err != nil {
		t.Fatalf("NewBindingService() error = %v", err)
	}
	if err := bindings.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("bindings AutoMigrate() error = %v", err)
	}

	project := testProject{
		Name:  "Project One",
		Token: "project-token",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project error = %v", err)
	}
	collection := testRAGCollection{
		CollectionID: "kb-1",
		Name:         "KB One",
		DataType:     "documents",
		ServiceMode:  "bleve_only",
		IndexPath:    filepath.Join(tempDir, "bleve", "kb-1"),
	}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("create collection error = %v", err)
	}
	link := testProjectRAGCollection{
		ProjectID:       project.ID,
		RAGCollectionID: collection.ID,
	}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("create link error = %v", err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(runtime, mux, authService, nil, ragboxService, bindings, nil, nil, nil, nil)

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/pro/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password123","expires_in_days":30}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	mux.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body=%s", loginResponse.Code, http.StatusOK, loginResponse.Body.String())
	}

	var loginPayload proLoginResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("json.Unmarshal(login) error = %v", err)
	}
	if loginPayload.Token == "" {
		t.Fatal("expected login token")
	}

	archiveBody := buildKnowledgeBaseZipBody(t)
	uploadRequest := httptest.NewRequest(http.MethodPost, "/api/pro/ragbox/collections/kb-1/files:upload", archiveBody.reader)
	uploadRequest.Header.Set("Content-Type", archiveBody.contentType)
	uploadRequest.Header.Set("Authorization", "Bearer "+loginPayload.Token)
	uploadResponse := httptest.NewRecorder()
	mux.ServeHTTP(uploadResponse, uploadRequest)
	if uploadResponse.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want %d, body=%s", uploadResponse.Code, http.StatusAccepted, uploadResponse.Body.String())
	}

	var response proRagBoxWriteResponse
	if err := json.Unmarshal(uploadResponse.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal(upload) error = %v", err)
	}
	if !response.Local.Indexed {
		t.Fatal("expected local knowledge base to be indexed")
	}
	if !response.RagBox.Skipped {
		t.Fatal("expected ragbox upload to be skipped for bleve_only knowledge base")
	}
	if response.RagBox.Reason != "knowledge base does not use RagBox" {
		t.Fatalf("ragbox skip reason = %q", response.RagBox.Reason)
	}
	if len(response.Local.WrittenFiles) == 0 {
		t.Fatal("expected extracted files to be reported")
	}
}

type multipartBody struct {
	reader      io.Reader
	contentType string
}

func buildKnowledgeBaseZipBody(t *testing.T) multipartBody {
	t.Helper()

	archiveBytes := new(bytes.Buffer)
	archiveWriter := zip.NewWriter(archiveBytes)
	fileWriter, err := archiveWriter.Create("docs/readme.txt")
	if err != nil {
		t.Fatalf("zip.Create() error = %v", err)
	}
	if _, err := fileWriter.Write([]byte("hello from zip")); err != nil {
		t.Fatalf("zip write error = %v", err)
	}
	if err := archiveWriter.Close(); err != nil {
		t.Fatalf("archiveWriter.Close() error = %v", err)
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("file", "bundle.zip")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := io.Copy(filePart, bytes.NewReader(archiveBytes.Bytes())); err != nil {
		t.Fatalf("copy zip body error = %v", err)
	}
	if err := writer.WriteField("storage_mode", "keep"); err != nil {
		t.Fatalf("WriteField(storage_mode) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	return multipartBody{
		reader:      body,
		contentType: writer.FormDataContentType(),
	}
}

func mustSplitHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()

	parsedHost := rawURL
	if len(parsedHost) >= len("http://") && parsedHost[:len("http://")] == "http://" {
		parsedHost = parsedHost[len("http://"):]
	}
	host, portText, err := net.SplitHostPort(parsedHost)
	if err != nil {
		t.Fatalf("net.SplitHostPort(%q) error = %v", rawURL, err)
	}
	port, err := net.LookupPort("tcp", portText)
	if err != nil {
		t.Fatalf("net.LookupPort(%q) error = %v", portText, err)
	}
	return host, port
}

type testProject struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`
	RootPath    string `gorm:"size:1024"`
	Token       string `gorm:"size:64;uniqueIndex;not null"`
}

func (testProject) TableName() string {
	return "projects"
}

type testRAGCollection struct {
	ID                 uint   `gorm:"primaryKey"`
	CollectionID       string `gorm:"size:255;uniqueIndex;not null"`
	Name               string `gorm:"size:255;not null"`
	DataType           string `gorm:"size:64;not null"`
	SourcePath         string `gorm:"size:2048"`
	AutoReindex        bool   `gorm:"not null;default:false"`
	ServiceMode        string `gorm:"size:64;not null;default:'bleve_only'"`
	VectorConnectionID string `gorm:"size:255"`
	IndexPath          string `gorm:"size:2048;not null"`
}

func (testRAGCollection) TableName() string {
	return "rag_collections"
}

type testProjectRAGCollection struct {
	ProjectID       uint `gorm:"primaryKey"`
	RAGCollectionID uint `gorm:"primaryKey"`
}

func (testProjectRAGCollection) TableName() string {
	return "project_rag_collections"
}
