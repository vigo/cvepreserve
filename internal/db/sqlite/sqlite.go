/*
Package sqlite implements sqlite database operations.
*/
package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // sqlite embedded
	"github.com/vigo/cvepreserve/internal/dbmodel"
)

var _ DBManager = (*DB)(nil) // compile time proof

// sentinel errors.
var (
	ErrValueRequired = errors.New("value required")
)

// DBManager defines database behaviours.
type DBManager interface {
	InitDB() error
	Save(model *dbmodel.CVE) error
	FindPagesNeedingRender() (dbmodel.RenderRequiredCVES, error)
	UpdateRenderedHTML(id int, html string) error
	IsCompleted(id int, url string) (bool, error)
	MarkCompleted(id int) error
}

// DB holds sqlite related params.
type DB struct {
	DB                   *sql.DB
	TargetSqliteFilename string
}

// InitDB creates initial sqlite table.
func (d *DB) InitDB() error {
	query := `CREATE TABLE IF NOT EXISTS cve_pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		cve_id TEXT,
		url TEXT UNIQUE,
		wayback_url TEXT,
		html TEXT,
		js_required INTEGER DEFAULT 0,
		completed INTEGER DEFAULT 0,
		status_code INTEGER,
		headers JSON,
		UNIQUE (cve_id, url)
	);`
	_, err := d.DB.Exec(query)

	return err
}

// Save inserts data to db.
func (d *DB) Save(model *dbmodel.CVE) error {
	headersJSON, err := json.Marshal(model.Headers)
	if err != nil {
		return err
	}

	_, err = d.DB.Exec(
		"INSERT OR IGNORE INTO cve_pages (created_at, cve_id, url, wayback_url, html, js_required, completed, status_code, headers) VALUES (CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, ?, ?)",
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

// FindPagesNeedingRender queries the DB for pages that require rendering.
func (d *DB) FindPagesNeedingRender() (dbmodel.RenderRequiredCVES, error) {
	rows, err := d.DB.Query("SELECT id, url FROM cve_pages WHERE js_required = 1 AND completed = 0")
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

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
	_, err := d.DB.Exec("UPDATE cve_pages SET html = ? WHERE id = ?", html, id)
	return err
}

// IsCompleted checks if page is already rendered.
func (d *DB) IsCompleted(id int, url string) (bool, error) {
	var completed bool

	err := d.DB.QueryRow("SELECT completed FROM cve_pages WHERE id = ? AND url = ?", id, url).Scan(&completed)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return completed, nil
}

// MarkCompleted marks as page render completed.
func (d *DB) MarkCompleted(id int) error {
	_, err := d.DB.Exec("UPDATE cve_pages SET completed = 1 WHERE id = ?", id)
	return err
}

func (d *DB) setDefaults() {
	if d.TargetSqliteFilename == "" {
		d.TargetSqliteFilename = "dataset.json"
	}
}

// Option represents option function type.
type Option func(*DB) error

// WithTargetSqliteFilename sets sqlite filename for creation.
func WithTargetSqliteFilename(s string) Option {
	return func(d *DB) error {
		if s == "" {
			return fmt.Errorf("%w, target filename can not be empty string", ErrValueRequired)
		}

		d.TargetSqliteFilename = s

		return nil
	}
}

// New instantiates new database instance.
func New(options ...Option) (*DB, error) {
	db := new(DB)
	for _, option := range options {
		if err := option(db); err != nil {
			return nil, err
		}
	}

	db.setDefaults()

	sqliteDB, err := sql.Open("sqlite3", db.TargetSqliteFilename)
	if err != nil {
		return nil, err
	}
	db.DB = sqliteDB

	return db, nil
}
