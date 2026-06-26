package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/new-api-tools/backend/internal/config"
	_ "modernc.org/sqlite"
)

type LocalStore struct {
	db *sql.DB
}

var localStore *LocalStore

func initLocalStore(cfg *config.Config) error {
	path := filepath.Join(cfg.DataDir, "local.db")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create local data dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open local sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &LocalStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return err
	}
	localStore = store
	return nil
}

func GetLocalStore() *LocalStore {
	return localStore
}

func closeLocalStore() error {
	if localStore != nil && localStore.db != nil {
		return localStore.db.Close()
	}
	return nil
}

func (s *LocalStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS cache (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expire_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS security_audit (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			user_id INTEGER,
			details TEXT,
			ip_address TEXT,
			created_at INTEGER NOT NULL
		);
	`)
	return err
}

func (s *LocalStore) GetConfig(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, nil
	}
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *LocalStore) SetConfig(ctx context.Context, key, value string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO config (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, now)
	return err
}

func (s *LocalStore) DeleteConfig(ctx context.Context, key string) (bool, error) {
	if s == nil || s.db == nil {
		return false, nil
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM config WHERE key = ?`, key)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *LocalStore) AllConfig(ctx context.Context) (map[string]string, error) {
	result := map[string]string{}
	if s == nil || s.db == nil {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM config ORDER BY key`)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return result, err
		}
		result[key] = value
	}
	return result, rows.Err()
}
