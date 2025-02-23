package cve

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/vigo/cvepreserve/internal/db/sqlite"
	"github.com/vigo/cvepreserve/internal/dbmodel"
	"github.com/vigo/cvepreserve/internal/wayback"
)

// FetchResult holds the response body, status code, and headers from an HTTP request.
type FetchResult struct {
	Headers    http.Header
	Body       string
	StatusCode int
}

// FetchAndStore fetches URLs and saves them in the database concurrently.
func FetchAndStore(db *sqlite.DB, client *http.Client, data <-chan Element, workers int, logger *slog.Logger) {
	var wg sync.WaitGroup

	fetchChan := make(chan Element, workers)

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for item := range fetchChan {
				processItem(db, client, item, logger)
			}
		}()
	}

	for item := range data {
		fetchChan <- item
	}
	close(fetchChan)
	wg.Wait()
}

func processItem(db *sqlite.DB, client *http.Client, item Element, logger *slog.Logger) {
	var wg sync.WaitGroup
	resultChan := make(chan *dbmodel.CVE, len(item.URLS))

	for _, url := range item.URLS {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			crawled, err := isCrawled(db.DB, item.CVEID, url)
			if err != nil {
				logger.Error("isCrawled", "err", err)

				return
			}

			if crawled {
				logger.Info("already crawled, skipping", "url", url)

				return
			}

			fetchResult, err := fetchURL(client, url, logger)
			if err != nil {
				waybackURL, errr := wayback.Fetch(client, url)
				if errr != nil {
					logger.Error("wayback.Fetch", "err", err, "url", url)

					return
				}

				crawled, err = isCrawled(db.DB, item.CVEID, waybackURL)
				if err != nil {
					logger.Error("isCrawled-waybackURL", "err", err, "url", waybackURL)

					return
				}

				if crawled {
					logger.Info("already crawled, skipping waybackURL", "url", waybackURL)

					return
				}

				fetchResult, err = fetchURL(client, waybackURL, logger)
				if err != nil {
					logger.Error("fetchURL wayback", "err", err, "url", waybackURL)

					return
				}

				url = waybackURL
			}

			jsRequired := fetchResult.Body == "" || strings.Contains(strings.ToLower(fetchResult.Body), "<noscript>")
			completed := !jsRequired

			resultChan <- &dbmodel.CVE{
				CVEID:      item.CVEID,
				URL:        url,
				HTML:       fetchResult.Body,
				JSRequired: jsRequired,
				Completed:  completed,
				StatusCode: fetchResult.StatusCode,
				Headers:    fetchResult.Headers,
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for model := range resultChan {
		if err := db.Save(model); err != nil {
			logger.Error("db.Save", "err", err, "url", model.URL)
		}
	}
}

func fetchURL(client *http.Client, url string, logger *slog.Logger) (*FetchResult, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	logger.Debug("fetchURL", "url", url, "status code", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyString := string(body)
	logger.Debug("fetchURL", "url", url, "body len", len(bodyString))

	return &FetchResult{
		Body:       bodyString,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}, nil
}

func isCrawled(db *sql.DB, cveID, url string) (bool, error) {
	var exists bool

	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM cve_pages WHERE cve_id = ? AND url = ?)", cveID, url).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
