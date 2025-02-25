/*
Package postgresql implements PostgreSQL database operations.
*/
package postgresql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/vigo/cvepreserve/internal/db"
	"github.com/vigo/cvepreserve/internal/dbmodel"
)

var _ db.Manager = (*DB)(nil) // Compile-time check

// DB holds PostgreSQL related parameters.
type DB struct {
	*sql.DB
	DSN string
}

// InitDB creates the initial PostgreSQL table.
// You need to `createdb` manually!
func (d *DB) InitDB() error {
	query := `CREATE TABLE IF NOT EXISTS "cve_pages" (
		"id" SERIAL PRIMARY KEY,
		"created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		"cve_id" VARCHAR(128) NOT NULL,
		"url" TEXT NOT NULL UNIQUE,
		"wayback_url" TEXT,
		"html" TEXT,
		"js_required" BOOLEAN NOT NULL DEFAULT FALSE,
		"completed" BOOLEAN NOT NULL DEFAULT FALSE,
		"status_code" INTEGER,
		"headers" JSONB NOT NULL DEFAULT '{}',
		UNIQUE ("cve_id", "url")
	);`
	_, err := d.DB.Exec(query)
	return err
}

// GetDB returns the underlying sql.DB instance.
func (d *DB) GetDB() *sql.DB {
	return d.DB
}

// Save inserts data into the PostgreSQL database.
func (d *DB) Save(model *dbmodel.CVE) error {
	headersJSON, err := json.Marshal(model.Headers)
	if err != nil {
		return err
	}

	_, err = d.DB.Exec(
		`INSERT INTO cve_pages (cve_id, url, wayback_url, html, js_required, completed, status_code, headers)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (cve_id, url) DO NOTHING`,
		model.CVEID,
		model.URL,
		model.WaybackURL,
		model.HTML,
		model.JSRequired,
		model.Completed,
		model.StatusCode,
		headersJSON,
	)
	return err
}

// FindPagesNeedingRender queries for pages that require rendering.
func (d *DB) FindPagesNeedingRender() (dbmodel.RenderRequiredCVES, error) {
	rows, err := d.DB.Query("SELECT id, url FROM cve_pages WHERE js_required = TRUE AND completed = FALSE")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results dbmodel.RenderRequiredCVES
	for rows.Next() {
		var entry dbmodel.RenderRequiredCVE
		if err = rows.Scan(&entry.ID, &entry.URL); err != nil {
			return nil, err
		}
		results = append(results, entry)
	}

	return results, nil
}

// UpdateRenderedHTML updates the HTML field after rendering.
func (d *DB) UpdateRenderedHTML(id int, html string) error {
	result, err := d.DB.Exec("UPDATE cve_pages SET html = $1 WHERE id = $2", html, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return db.ErrNoRowsAffected
	}

	return nil
}

// IsCompleted checks if a page has already been rendered.
func (d *DB) IsCompleted(id int, url string) (bool, error) {
	var completed bool
	err := d.DB.QueryRow("SELECT completed FROM cve_pages WHERE id = $1 AND url = $2", id, url).Scan(&completed)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return completed, nil
}

// MarkCompleted marks a page as render completed.
func (d *DB) MarkCompleted(id int) error {
	result, err := d.DB.Exec("UPDATE cve_pages SET completed = TRUE WHERE id = $1", id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return db.ErrNoRowsAffected
	}

	return nil
}

// Option represents an option function type.
type Option func(*DB) error

// WithDSN sets the PostgreSQL DSN (Data Source Name).
func WithDSN(dsn string) Option {
	return func(d *DB) error {
		if dsn == "" {
			return fmt.Errorf("%w, dsn cannot be empty", db.ErrValueRequired)
		}

		d.DSN = dsn

		return nil
	}
}

// New initializes a new PostgreSQL database instance.
func New(options ...Option) (*DB, error) {
	dbase := new(DB)
	for _, option := range options {
		if err := option(dbase); err != nil {
			return nil, err
		}
	}

	if dbase.DSN == "" {
		dbase.DSN = os.Getenv("DATABASE_URL")
	}
	if dbase.DSN == "" {
		return nil, fmt.Errorf("%w, dsn cannot be empty", db.ErrValueRequired)
	}

	// Connect to PostgreSQL using DSN
	pgDB, err := sql.Open("postgres", dbase.DSN)
	if err != nil {
		return nil, err
	}
	dbase.DB = pgDB

	return dbase, nil
}
