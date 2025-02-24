package cve

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/vigo/cvepreserve/internal/db"
	"github.com/vigo/cvepreserve/internal/dbmodel"
	"github.com/vigo/cvepreserve/internal/httpclient"
	"github.com/vigo/cvepreserve/internal/wayback"
)

// FetchResult holds the response body, status code, and headers from an HTTP request.
type FetchResult struct {
	Headers    http.Header
	Body       string
	StatusCode int
}

// FetchAndStore fetches URLs and saves them in the database concurrently.
func FetchAndStore(dbase db.Manager, cl httpclient.Doer, data <-chan Element, workers int, logger *slog.Logger) {
	var wg sync.WaitGroup

	fetchChan := make(chan Element, workers)
	logger.Debug("starting FetchAndStore")

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for item := range fetchChan {
				processItem(dbase, cl, item, logger)
			}
		}()
	}

	for item := range data {
		fetchChan <- item
	}
	close(fetchChan)
	wg.Wait()
}

func processItem(dbase db.Manager, cl httpclient.Doer, item Element, logger *slog.Logger) {
	var wg sync.WaitGroup
	resultChan := make(chan *dbmodel.CVE, len(item.URLS))

	logger.Debug("processItem", "CVEID", item.CVEID)

	for _, url := range item.URLS {
		wg.Add(1)

		go func() {
			defer wg.Done()

			crawled, err := isCrawled(dbase, item.CVEID, url)
			if err != nil {
				logger.Error("isCrawled", "err", err)

				return
			}

			if crawled {
				logger.Info("already crawled, skipping", "url", url)

				return
			}

			possibleWaybackURL := ""
			fetchResult, err := fetchURL(cl, url, logger)
			if err != nil {
				waybackURL, errr := wayback.Fetch(cl, url)
				if errr != nil {
					logger.Error("wayback.Fetch", "err", err, "url", url)

					return
				}

				fetchResult, err = fetchURL(cl, waybackURL, logger)
				if err != nil {
					logger.Error("fetchURL wayback", "err", err, "url", waybackURL)

					return
				}

				possibleWaybackURL = waybackURL
			}

			jsRequired := fetchResult.Body == "" || strings.Contains(strings.ToLower(fetchResult.Body), "<noscript>")
			completed := !jsRequired

			resultChan <- &dbmodel.CVE{
				CVEID:      item.CVEID,
				URL:        url,
				WaybackURL: possibleWaybackURL,
				HTML:       fetchResult.Body,
				JSRequired: jsRequired,
				Completed:  completed,
				StatusCode: fetchResult.StatusCode,
				Headers:    fetchResult.Headers,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for model := range resultChan {
		if err := dbase.Save(model); err != nil {
			logger.Error("db.Save", "err", err, "url", model.URL)
		}
	}
}

func fetchURL(cl httpclient.Doer, url string, logger *slog.Logger) (*FetchResult, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", httpclient.UserAgent)

	resp, err := cl.Do(req)
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

func isCrawled(dbase db.Manager, cveID, url string) (bool, error) {
	var exists bool

	db := dbase.GetDB()
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM cve_pages WHERE cve_id = ? AND url = ?)", cveID, url).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
