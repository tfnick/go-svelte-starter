package db

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/tfnick/sqlx"
	"github.com/tfnick/go-svelte-starter/api/framework/logging"
	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

//go:embed migrations/app/*.sql
var appMigrationsFS embed.FS

//go:embed migrations/shared/*.sql
var sharedMigrationsFS embed.FS

type namedDB struct {
	db     *sqlx.DB
	driver string
	path   string
}

type DBManager struct {
	databases map[string]*namedDB
	mu        sync.RWMutex
}

var logger = logging.For("db")

func NewDBManager() *DBManager {
	return &DBManager{
		databases: make(map[string]*namedDB),
	}
}

func sqlitePragmas() []struct{ sql, desc string } {
	return []struct{ sql, desc string }{
		{"PRAGMA foreign_keys = ON", "foreign key constraints"},
		{"PRAGMA journal_mode = WAL", "WAL mode"},
		{"PRAGMA synchronous = NORMAL", "sync mode"},
		{"PRAGMA cache_size = -64000", "64MB cache"},
		{"PRAGMA temp_store = MEMORY", "memory temp store"},
	}
}

func applyConfig(d *sqlx.DB, driver string) error {
	switch driver {
	case "sqlite":
		d.SetMaxOpenConns(1)
		d.SetMaxIdleConns(1)
		for _, p := range sqlitePragmas() {
			if _, err := d.Exec(p.sql); err != nil {
				return fmt.Errorf("set %s failed: %w", p.desc, err)
			}
		}
	case "postgres":
		d.SetMaxOpenConns(25)
		d.SetMaxIdleConns(5)
	default:
		return fmt.Errorf("unsupported database driver: %s", driver)
	}
	return nil
}

func openConfiguredDB(name, driver, path string) (*sqlx.DB, error) {
	db, err := sqlx.Open(driver, path)
	if err != nil {
		return nil, fmt.Errorf("open database %s failed: %w", name, err)
	}

	if err := applyConfig(db, driver); err != nil {
		db.Close()
		return nil, fmt.Errorf("configure database %s failed: %w", name, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database %s failed: %w", name, err)
	}

	return db, nil
}

func (m *DBManager) Open(name, driver, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.databases[name]; exists {
		return fmt.Errorf("database already registered: %s", name)
	}

	db, err := openConfiguredDB(name, driver, path)
	if err != nil {
		return err
	}

	m.databases[name] = &namedDB{
		db:     db,
		driver: driver,
		path:   path,
	}

	logger.Info().Str("database", name).Str("path", path).Msg("database connected")
	return nil
}

func (m *DBManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, ndb := range m.databases {
		if err := ndb.db.Close(); err != nil {
			lastErr = fmt.Errorf("close database %s failed: %w", name, err)
		}
	}

	m.databases = make(map[string]*namedDB)
	return lastErr
}

func (m *DBManager) GetDB(name string) (*sqlx.DB, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ndb, ok := m.databases[name]
	if !ok {
		return nil, fmt.Errorf("database not found: %s", name)
	}
	return ndb.db, nil
}

func (m *DBManager) GetEngine(name string) (*sqlx.Engine, error) {
	db, err := m.GetDB(name)
	if err != nil {
		return nil, err
	}
	return db.LazyEngine(), nil
}

type Transaction func(*sqlx.Tx) error

func (m *DBManager) WithTransaction(name string, fn Transaction) error {
	db, err := m.GetDB(name)
	if err != nil {
		return err
	}
	return db.WithTransaction(fn)
}

func (m *DBManager) AutoMigrate(name string) error {
	db, err := m.GetDB(name)
	if err != nil {
		return err
	}

	var migrationsFS embed.FS
	switch name {
	case "app":
		migrationsFS = appMigrationsFS
	case "shared":
		migrationsFS = sharedMigrationsFS
	default:
		return fmt.Errorf("unknown database: %s", name)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations failed: %w", err)
	}

	var applied []string
	if err := db.Select(&applied, "SELECT name FROM schema_migrations"); err != nil {
		return fmt.Errorf("query applied migrations failed: %w", err)
	}

	appliedMap := make(map[string]bool, len(applied))
	for _, v := range applied {
		appliedMap[v] = true
	}

	dir := "migrations/" + name
	files, err := migrationsFS.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s failed: %w", dir, err)
	}

	var sortedFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".sql" {
			sortedFiles = append(sortedFiles, f.Name())
		}
	}
	sort.Strings(sortedFiles)

	insertSQL := db.Rebind(`INSERT INTO schema_migrations (name) VALUES (?)`)
	migrated := false
	for _, filename := range sortedFiles {
		if appliedMap[filename] {
			logger.Info().Str("database", name).Str("migration", filename).Msg("skip applied migration")
			continue
		}

		sqlContent, err := migrationsFS.ReadFile(dir + "/" + filename)
		if err != nil {
			return fmt.Errorf("read migration %s failed: %w", filename, err)
		}

		if _, err := db.Exec(string(sqlContent)); err != nil {
			return fmt.Errorf("execute migration %s failed: %w", filename, err)
		}

		if _, err := db.Exec(insertSQL, filename); err != nil {
			return fmt.Errorf("record migration %s failed: %w", filename, err)
		}

		logger.Info().Str("database", name).Str("migration", filename).Msg("applied migration")
		migrated = true
	}

	if migrated {
		logger.Info().Str("database", name).Msg("database migrations complete")
	} else {
		logger.Info().Str("database", name).Msg("database already up to date")
	}

	return nil
}

func (m *DBManager) Reopen(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ndb, ok := m.databases[name]
	if !ok {
		return fmt.Errorf("database not found: %s", name)
	}

	db, err := openConfiguredDB(name, ndb.driver, ndb.path)
	if err != nil {
		return err
	}

	oldDB := ndb.db
	ndb.db = db
	if oldDB != nil {
		if err := oldDB.Close(); err != nil {
			ndb.db = oldDB
			db.Close()
			return fmt.Errorf("close old database connection failed: %w", err)
		}
	}

	logger.Info().Str("database", name).Msg("database reloaded")
	return nil
}

func EnsureDataDir() error {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		return os.MkdirAll("data", 0755)
	}
	return nil
}

var DefaultManager *DBManager

func GetDB(name string) (*sqlx.DB, error) {
	return DefaultManager.GetDB(name)
}

func GetEngine(name string) (*sqlx.Engine, error) {
	return DefaultManager.GetEngine(name)
}

func WithTransaction(name string, fn Transaction) error {
	return DefaultManager.WithTransaction(name, fn)
}

func AutoMigrateDB(name string) error {
	return DefaultManager.AutoMigrate(name)
}

func ReopenDB(name string) error {
	return DefaultManager.Reopen(name)
}
