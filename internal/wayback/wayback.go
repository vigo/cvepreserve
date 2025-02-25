/*
Package wayback implements page fecth functionality.
*/
package wayback

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/vigo/cvepreserve/internal/httpclient"
)

// sentinel errors.
var (
	ErrSnapshotNotFound = errors.New("wayback snapshot not found")
)

const waybackCDXEndpoint = "https://web.archive.org/cdx/search/cdx?url=%s&output=json&filter=statuscode:200&limit=1&sort=timestamp"

// Fetch queries Wayback Machine for the earliest archived version of a URL.
func Fetch(cl httpclient.Doer, url string) (string, error) {
	requestURL := fmt.Sprintf(waybackCDXEndpoint, url)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", httpclient.UserAgent)

	resp, err := cl.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var data [][]string
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode Wayback response: %w", err)
	}

	if len(data) < 2 || len(data[1]) < 3 {
		return "", ErrSnapshotNotFound
	}

	timestamp := data[1][1]
	archiveURL := fmt.Sprintf("https://web.archive.org/web/%s/%s", timestamp, url)

	return archiveURL, nil
}
