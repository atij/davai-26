package db

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/adoreme/geo-tracker/internal/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

//go:embed schema.sql
var schemaSQL string

func Connect(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	return db, nil
}

func Migrate(db *sqlx.DB) error {
	// Simple idempotent migration
	queries := strings.Split(schemaSQL, ";")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		// MySQL doesn't support CREATE INDEX IF NOT EXISTS before 8.0.30
		// We handle the error if the index already exists
		if _, err := db.Exec(query); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key name") {
				continue
			}
			return fmt.Errorf("exec query: %s: %w", query, err)
		}
	}
	return nil
}

func Reset(db *sqlx.DB) error {
	// 1. Disable foreign key checks
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("disable fk checks: %w", err)
	}
	defer db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// 2. Get all tables in the current database
	var tables []string
	err := db.Select(&tables, "SHOW TABLES")
	if err != nil {
		return fmt.Errorf("show tables: %w", err)
	}

	// 3. Drop all tables
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return fmt.Errorf("drop table %s: %w", table, err)
		}
	}

	// 4. Run migrations to recreate schema
	return Migrate(db)
}
