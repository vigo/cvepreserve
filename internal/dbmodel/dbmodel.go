/*
Package dbmodel defines db model structure.
*/
package dbmodel

import "net/http"

// CVE represents `cve` table and fields.
type CVE struct {
	Headers    http.Header
	CVEID      string
	URL        string
	HTML       string
	StatusCode int
	JSRequired bool
	Completed  bool
}

// RenderRequiredCVE represents lookup item.
type RenderRequiredCVE struct {
	URL string
	ID  int
}

// RenderRequiredCVES is a collection of RenderRequiredCVEs.
type RenderRequiredCVES []RenderRequiredCVE
