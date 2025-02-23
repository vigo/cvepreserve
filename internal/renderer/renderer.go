/*
Package renderer implements JavaScript required page's operations.
*/
package renderer

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/vigo/cvepreserve/internal/db/sqlite"
	"github.com/vigo/cvepreserve/internal/dbmodel"
)

const waitTime = 3 * time.Second

// RenderRequiredPages finds and renders pages requiring JavaScript execution.
func RenderRequiredPages(db *sqlite.DB, workers int, logger *slog.Logger) {
	pages, err := db.FindPagesNeedingRender()
	if err != nil {
		logger.Error("db.FindPagesNeedingRender", "err", err)
		return
	}

	if len(pages) == 0 {
		logger.Info("no pages need rendering")
		return
	}

	logger.Info("found", "page(s)", len(pages))

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllocator()

	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	renderChan := make(chan dbmodel.RenderRequiredCVE, len(pages))

	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for job := range renderChan {
				completed, errr := db.IsCompleted(job.ID, job.URL)
				if errr != nil {
					logger.Error("db.IsCompleted", "err", errr, "ID", job.ID, "url", job.URL, "worker", i)

					continue
				}

				if completed {
					logger.Info("completed", "ID", job.ID, "url", job.URL, "worker", i)

					continue
				}

				logger.Info("render", "url", job.URL)

				html, errr := renderPage(browserCtx, job.URL, logger)
				if errr != nil {
					logger.Error("renderPage", "err", errr, "url", job.URL, "worker", i)

					continue
				}

				errr = db.UpdateRenderedHTML(job.ID, html)
				if errr != nil {
					logger.Error("UpdateRenderedHTML", "err", errr, "ID", job.ID, "url", job.URL, "worker", i)

					continue
				}

				if errrr := db.MarkCompleted(job.ID); errrr != nil {
					logger.Error("db.MarkCompleted", "err", errrr, "ID", job.ID, "url", job.URL, "worker", i)
				}
			}
		}()
	}

	for _, page := range pages {
		renderChan <- page
	}
	close(renderChan)

	wg.Wait()
}

func renderPage(parentCtx context.Context, url string, logger *slog.Logger) (string, error) {
	ctx, cancel := chromedp.NewContext(parentCtx)
	defer cancel()

	var html string

	logger.Debug("navigating to", "url", url)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(waitTime),
		chromedp.OuterHTML("html", &html),
	); err != nil {
		logger.Error("renderPage", "err", err, "url", url)

		return "", err
	}

	if html == "" {
		logger.Warn("empty html received", "url", url)
	}

	return html, nil
}
