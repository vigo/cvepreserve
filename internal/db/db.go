/*
Package db provides database abstraction.
*/
package db

import (
	"database/sql"
	"errors"

	"github.com/vigo/cvepreserve/internal/dbmodel"
)

// Manager defines database behaviours.
type Manager interface {
	InitDB() error
	GetDB() *sql.DB
	Save(model *dbmodel.CVE) error
	FindPagesNeedingRender() (dbmodel.RenderRequiredCVES, error)
	UpdateRenderedHTML(id int, html string) error
	IsCompleted(id int, url string) (bool, error)
	MarkCompleted(id int) error
}

// sentinel errors.
var (
	ErrValueRequired  = errors.New("value required")
	ErrNoRowsAffected = errors.New("no row(s) affected")
)
