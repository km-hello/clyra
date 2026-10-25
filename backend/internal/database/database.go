package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := (&url.URL{Scheme: "file", Path: absolute}).String() + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=FULL&_txlock=immediate"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = initializeSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func initializeSchema(db *sql.DB) error {
	var tables, versionTables int
	err := db.QueryRow(`SELECT count(*),count(CASE WHEN name='goose_db_version' THEN 1 END) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables, &versionTables)
	if err != nil {
		return err
	}
	if tables > 0 && versionTables == 0 {
		return fmt.Errorf("database has no Goose migration history; reset the database before starting this version of Clyra")
	}
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, files)
	if err != nil {
		return err
	}
	ctx := context.Background()
	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return err
	}
	if current > target {
		return fmt.Errorf("database version %d is newer than supported version %d", current, target)
	}
	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}
