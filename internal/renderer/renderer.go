/*
Package renderer implements JavaScript required page's operations.
*/
package renderer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/vigo/cvepreserve/internal/colorz"
	"github.com/vigo/cvepreserve/internal/db/sqlite"
	"github.com/vigo/cvepreserve/internal/dbmodel"
)

const waitTime = 3 * time.Second

// RenderRequiredPages finds and renders pages requiring JavaScript execution.
func RenderRequiredPages(db *sqlite.DB, workers int) {
	pages, err := db.FindPagesNeedingRender()
	if err != nil {
		fmt.Println(colorz.Red+"[error][db.FindPagesNeedingRender]", err, colorz.Reset)
		return
	}

	if len(pages) == 0 {
		fmt.Println(colorz.White+"[info] no pages need rendering", colorz.Reset)
		return
	}

	fmt.Printf(colorz.White+"[info] found %d pages for rendering: %v\n"+colorz.Reset, len(pages), pages)

	renderChan := make(chan dbmodel.RenderRequiredCVE, len(pages))

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for job := range renderChan {
				completed, errr := db.IsCompleted(job.ID, job.URL)
				if errr != nil {
					fmt.Println(colorz.Red+"[error][db.IsCompleted]", errr, job.ID, job.URL, colorz.Reset)

					continue
				}

				if completed {
					fmt.Println(colorz.White+"[info][completed]", job.ID, job.URL, colorz.Reset)

					continue
				}

				fmt.Println(colorz.White+"[info][render]", job.URL, colorz.Reset)

				html, errr := renderPage(job.URL)
				if errr != nil {
					fmt.Println(colorz.Red+"[error][renderPage]", errr, job.URL, colorz.Reset)

					continue
				}

				errr = db.UpdateRenderedHTML(job.ID, html)
				if errr != nil {
					fmt.Println(colorz.Red+"[error][UpdateRenderedHTML]", errr, job.ID, job.URL, colorz.Reset)

					continue
				}

				if errrr := db.MarkCompleted(job.ID); errrr != nil {
					fmt.Println(colorz.Red+"[error][db.MarkCompleted]", errrr, job.ID, job.URL, colorz.Reset)
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

func renderPage(url string) (string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var html string

	fmt.Println(colorz.Yellow+"[debug] navigating to:", url, colorz.Reset)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(waitTime),
		chromedp.OuterHTML("html", &html),
	); err != nil {
		fmt.Println(colorz.Red+"[error][renderPage]", err, url, colorz.Reset)

		return "", err
	}

	if html == "" {
		fmt.Println(colorz.White+"[info][empty-html]", url, colorz.Reset)
	}

	return html, nil
}
