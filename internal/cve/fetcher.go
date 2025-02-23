package cve

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/vigo/cvepreserve/internal/colorz"
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
func FetchAndStore(db *sqlite.DB, client *http.Client, data <-chan Element, workers int) {
	var wg sync.WaitGroup

	fetchChan := make(chan Element, workers)

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for item := range fetchChan {
				processItem(db, client, item)
			}
		}()
	}

	for item := range data {
		fetchChan <- item
	}
	close(fetchChan)
	wg.Wait()
}

func processItem(db *sqlite.DB, client *http.Client, item Element) {
	var wg sync.WaitGroup
	resultChan := make(chan *dbmodel.CVE, len(item.URLS))

	for _, url := range item.URLS {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			crawled, err := isCrawled(db.DB, item.CVEID, url)
			if err != nil {
				fmt.Println(colorz.Red+"[error][isCrawled]", err, url, colorz.Reset)

				return
			}

			if crawled {
				fmt.Println(colorz.White+"[info][skipping]", url, "(already crawled)", colorz.Reset)

				return
			}

			fetchResult, err := fetchURL(client, url)
			if err != nil {
				waybackURL, errr := wayback.Fetch(client, url)
				if errr != nil {
					fmt.Println(colorz.Red+"[error][wayback.Fetch]", err, url, colorz.Reset)

					return
				}

				crawled, err = isCrawled(db.DB, item.CVEID, waybackURL)
				if err != nil {
					fmt.Println(colorz.Red+"[error][isCrawled-waybackURL]", err, waybackURL, colorz.Reset)

					return
				}

				if crawled {
					fmt.Println(
						colorz.White+"[info][skipping-waybackURL]",
						waybackURL,
						"(already crawled)",
						colorz.Reset,
					)

					return
				}

				fetchResult, err = fetchURL(client, waybackURL)
				if err != nil {
					fmt.Println(colorz.Red+"[error][fetchURL-wayback]", err, waybackURL, colorz.Reset)

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
			fmt.Println(colorz.Red+"[error][db.Save]", err, model.URL, colorz.Reset)
		}
	}
}

func fetchURL(client *http.Client, url string) (*FetchResult, error) {
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

	fmt.Println(colorz.Yellow+"[debug][fetchURL]", url, "[status]", resp.StatusCode, colorz.Reset)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyString := string(body)
	var bodyStringSample string
	if len(bodyString) > 20 {
		bodyStringSample = bodyString[:20]
	}
	fmt.Println(
		colorz.Yellow+"[debug][fetchURL]",
		url,
		"[bodyString]",
		len(bodyString),
		"[bodyStringSample]",
		bodyStringSample,
		colorz.Reset,
	)

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
