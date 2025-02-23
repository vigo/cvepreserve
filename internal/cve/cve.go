/*
Package cve implements cve related functionality.
*/
package cve

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/vigo/cvepreserve/internal/colorz"
)

// Element represents CVE item.
type Element struct {
	CVEID string   `json:"cve_id"`
	URLS  []string `json:"urls"`
}

// Elements is a collection of Elements.
type Elements []Element

// FilterFunc represents filtering function.
type FilterFunc func(<-chan Element) <-chan Element

// ReadDataset reads dataset json file.
func ReadDataset(r io.Reader) chan Element {
	out := make(chan Element)

	go func() {
		defer close(out)

		decoder := json.NewDecoder(r)
		if _, err := decoder.Token(); err != nil {
			fmt.Println(colorz.Red+"[error][decoder-token]", err, colorz.Reset)
			return
		}

		for decoder.More() {
			var element Element
			if err := decoder.Decode(&element); err != nil {
				fmt.Println(colorz.Red+"[error][decoder-decode]", err, colorz.Reset)
				return
			}

			out <- element
		}
	}()
	return out
}

// FilterEmptyURLS filters empty url slices.
func FilterEmptyURLS(elements <-chan Element) <-chan Element {
	out := make(chan Element)

	go func() {
		defer close(out)

		for element := range elements {
			if len(element.URLS) > 0 {
				out <- element
			}
		}
	}()

	return out
}

// FanIn merges multiple read-only channels (`<-chan Element`) into a single
// output channel.
func FanIn(chans ...<-chan Element) <-chan Element {
	out := make(chan Element)

	var wg sync.WaitGroup
	wg.Add(len(chans))

	for _, ch := range chans {
		go func() {
			for element := range ch {
				out <- element
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
