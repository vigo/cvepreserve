/*
Package wayback implements page fecth functionality.
*/
package wayback

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Response represents Wayback response.
type Response struct {
	ArchivedSnapshots struct {
		Closest struct {
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
			URL       string `json:"url"`
			Available bool   `json:"available"`
		} `json:"closest"`
	} `json:"archived_snapshots"`
}

const waybackEndpoint = "https://archive.org/wayback/available?url="

// sentinel errors.
var (
	ErrSnapshotNotFound = errors.New("wayback snapshot not found")
)

// Fetch queries Wayback Machine for an archived version of a URL.
func Fetch(client *http.Client, url string) (string, error) {
	resp, err := client.Get(waybackEndpoint + url)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var wbResponse Response

	if err = json.NewDecoder(resp.Body).Decode(&wbResponse); err != nil {
		return "", err
	}

	if wbResponse.ArchivedSnapshots.Closest.Available && wbResponse.ArchivedSnapshots.Closest.Status == "200" {
		return wbResponse.ArchivedSnapshots.Closest.URL, nil
	}

	return "", ErrSnapshotNotFound
}
