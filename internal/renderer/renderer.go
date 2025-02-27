/*
Package renderer implements JavaScript required page's operations.
*/
package renderer

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/vigo/cvepreserve/internal/db"
	"github.com/vigo/cvepreserve/internal/dbmodel"
	"github.com/vigo/cvepreserve/internal/httpclient"
)

const (
	waitTime      = 3 * time.Second
	chromeTimeout = 10 * time.Second
)

// RenderRequiredPages finds and renders pages requiring JavaScript execution.
func RenderRequiredPages(dbase db.Manager, workers int, logger *slog.Logger) {
	pages, err := dbase.FindPagesNeedingRender()
	if err != nil {
		logger.Error("FindPagesNeedingRender", "err", err)
		return
	}

	if len(pages) == 0 {
		logger.Info("no pages need rendering")
		return
	}

	logger.Info("found", "page(s)", len(pages))

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-http2", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-site-isolation-trials", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.UserAgent(httpclient.UserAgent),
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
				completed, errr := dbase.IsCompleted(job.ID, job.URL)
				if errr != nil {
					logger.Error("IsCompleted", "err", errr, "ID", job.ID, "url", job.URL, "worker", i)

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

					if !errors.Is(errr, context.DeadlineExceeded) {
						continue
					}
				}

				errr = dbase.UpdateRenderedHTML(job.ID, html)
				if errr != nil {
					logger.Error("UpdateRenderedHTML", "err", errr, "ID", job.ID, "url", job.URL, "worker", i)

					continue
				}

				if errrr := dbase.MarkCompleted(job.ID); errrr != nil {
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

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, chromeTimeout)
	defer timeoutCancel()

	var html string

	logger.Debug("navigating to", "url", url)

	done := make(chan struct{})
	errChan := make(chan error, 1)

	go func() {
		err := chromedp.Run(
			timeoutCtx,
			chromedp.Navigate(url),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Sleep(waitTime),
			chromedp.OuterHTML("html", &html),
		)
		errChan <- err
		close(done)
	}()

	select {
	case <-done:
		err := <-errChan
		if html == "" {
			logger.Warn("empty html received", "url", url, "err", err)
		}
		return html, err
	case <-timeoutCtx.Done():
		err := <-errChan
		logger.Warn("chrome render timeouts", "url", url, "err", err, "html size", len(html))
		return html, err
	}
}
