package prostorage

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ArturSaleev/MCPBox/app"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultProbeTimeout = 3 * time.Second

var openAndPingPostgresFunc = openAndPingPostgres

type Manager struct {
	bootstrapDB *gorm.DB
	service     *Service

	mu             sync.RWMutex
	operationalDB  *gorm.DB
	operationalDSN string
	status         RuntimeStatus
}

func NewManager(runtime *app.RuntimeContext, service *Service) (*Manager, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	if service == nil {
		return nil, errors.New("storage config service is not available")
	}

	return &Manager{
		bootstrapDB:   runtime.DB,
		service:       service,
		operationalDB: runtime.DB,
		status: RuntimeStatus{
			BootstrapBackend:          backendSQLite,
			DesiredOperationalBackend: backendSQLite,
			ActiveOperationalBackend:  backendSQLite,
			DesiredLogsBackend:        logsBackendOperational,
			ActiveLogsBackend:         logsBackendOperational,
		},
	}, nil
}

func (m *Manager) Reload(ctx context.Context) (RuntimeStatus, error) {
	if m == nil || m.service == nil {
		return RuntimeStatus{}, errors.New("storage runtime manager is not initialized")
	}

	record, err := m.service.ensureRecord(ctx)
	if err != nil {
		return RuntimeStatus{}, err
	}

	status := RuntimeStatus{
		BootstrapBackend:          backendSQLite,
		DesiredOperationalBackend: normalizeOperationalBackend(record.OperationalBackend),
		ActiveOperationalBackend:  backendSQLite,
		DesiredLogsBackend:        normalizeLogsBackend(record.LogsBackend),
		ActiveLogsBackend:         logsBackendOperational,
	}

	operationalDB := m.bootstrapDB
	operationalDSN := ""

	postgresStatus := ConnectionStatus{
		Configured: postgresConfigConfigured(record),
	}
	if status.DesiredOperationalBackend == backendPostgres {
		if !postgresStatus.Configured {
			status.OperationalFallbackReason = "postgres backend is selected but postgres config is incomplete"
		} else {
			desiredDSN := buildPostgresDSN(record)
			if reusedDB, ok := m.reuseOperationalPostgres(ctx, desiredDSN); ok {
				postgresStatus.Reachable = true
				postgresStatus.Message = "ok"
				operationalDB = reusedDB
				operationalDSN = desiredDSN
				status.ActiveOperationalBackend = backendPostgres
			} else {
				db, pingErr := openAndPingPostgresFunc(ctx, desiredDSN)
				if pingErr != nil {
					postgresStatus.Message = pingErr.Error()
					status.OperationalFallbackReason = "postgres connection failed, bootstrap sqlite remains active"
				} else {
					postgresStatus.Reachable = true
					postgresStatus.Message = "ok"
					operationalDB = db
					operationalDSN = desiredDSN
					status.ActiveOperationalBackend = backendPostgres
				}
			}
		}
	}
	status.Postgres = postgresStatus

	status.ClickHouse = probeHTTPBackend(ctx, strings.TrimSpace(record.ClickHouseURL), http.MethodGet, "/ping")
	status.Elasticsearch = probeHTTPBackend(ctx, strings.TrimSpace(record.ElasticsearchURL), http.MethodGet, "")

	switch status.DesiredLogsBackend {
	case logsBackendClickHouse:
		if status.ClickHouse.Configured && status.ClickHouse.Reachable {
			status.ActiveLogsBackend = logsBackendClickHouse
		} else {
			status.ActiveLogsBackend = logsBackendOperational
			status.LogsFallbackReason = "clickhouse is not reachable, logs fallback to operational backend"
		}
	case logsBackendElastic:
		if status.Elasticsearch.Configured && status.Elasticsearch.Reachable {
			status.ActiveLogsBackend = logsBackendElastic
		} else {
			status.ActiveLogsBackend = logsBackendOperational
			status.LogsFallbackReason = "elasticsearch is not reachable, logs fallback to operational backend"
		}
	default:
		status.ActiveLogsBackend = logsBackendOperational
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.operationalDB != nil && m.operationalDB != m.bootstrapDB && m.operationalDB != operationalDB {
		if sqlDB, dbErr := m.operationalDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	}
	m.operationalDB = operationalDB
	m.operationalDSN = operationalDSN
	m.status = status
	return status, nil
}

func (m *Manager) Status() RuntimeStatus {
	if m == nil {
		return RuntimeStatus{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) OperationalDB() *gorm.DB {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.operationalDB == nil {
		return m.bootstrapDB
	}
	return m.operationalDB
}

func openAndPingPostgres(ctx context.Context, dsn string) (*gorm.DB, error) {
	probeCtx, cancel := context.WithTimeout(ctx, defaultProbeTimeout)
	defer cancel()

	db, err := gorm.Open(postgres.Open(strings.TrimSpace(dsn)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetConnMaxLifetime(time.Minute)
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(1)

	if err := sqlDB.PingContext(probeCtx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func (m *Manager) reuseOperationalPostgres(ctx context.Context, desiredDSN string) (*gorm.DB, bool) {
	if m == nil {
		return nil, false
	}

	m.mu.RLock()
	currentDB := m.operationalDB
	currentDSN := m.operationalDSN
	currentStatus := m.status
	m.mu.RUnlock()

	if currentDB == nil || currentDB == m.bootstrapDB {
		return nil, false
	}
	if currentStatus.ActiveOperationalBackend != backendPostgres {
		return nil, false
	}
	if strings.TrimSpace(currentDSN) != strings.TrimSpace(desiredDSN) {
		return nil, false
	}

	sqlDB, err := currentDB.DB()
	if err != nil {
		return nil, false
	}

	probeCtx, cancel := context.WithTimeout(ctx, defaultProbeTimeout)
	defer cancel()
	if err := sqlDB.PingContext(probeCtx); err != nil {
		return nil, false
	}
	return currentDB, true
}

func probeHTTPBackend(ctx context.Context, rawURL, method, path string) ConnectionStatus {
	rawURL = strings.TrimSpace(rawURL)
	status := ConnectionStatus{Configured: rawURL != ""}
	if rawURL == "" {
		return status
	}

	target := strings.TrimRight(rawURL, "/") + path
	probeCtx, cancel := context.WithTimeout(ctx, defaultProbeTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(probeCtx, method, target, nil)
	if err != nil {
		status.Message = err.Error()
		return status
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 400 {
		status.Reachable = true
		status.Message = "ok"
		return status
	}
	status.Message = "unexpected status"
	return status
}
