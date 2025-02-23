/*
Package main implements command-line functionality.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/vigo/cvepreserve/internal/colorz"
	"github.com/vigo/cvepreserve/internal/cve"
	"github.com/vigo/cvepreserve/internal/db/sqlite"
	"github.com/vigo/cvepreserve/internal/httpclient"
	"github.com/vigo/cvepreserve/internal/renderer"
	"github.com/vigo/cvepreserve/internal/version"
)

const (
	defaultShowVersion = false
	defaultDatasetFile = "dataset.json"
)

func main() {
	vrs := flag.Bool("version", defaultShowVersion, "display version information")
	dataset := flag.String("dataset", defaultDatasetFile, "dataset json filename")
	workers := flag.Int("workers", runtime.NumCPU(), "number of concurrent workers")
	flag.Parse()

	if *vrs {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", version.Version)
		return
	}

	f, err := os.Open(*dataset)
	if err != nil {
		fmt.Println(colorz.Red+"[error][dataset-open]", err, colorz.Reset)
		return
	}

	db, err := sqlite.New(
		sqlite.WithTargetSqliteFilename("result.sqlite3"),
	)
	if err != nil {
		fmt.Println(colorz.Red+"[error][db]", err, colorz.Reset)
		return
	}

	defer func() {
		_ = db.DB.Close()
	}()

	if err = db.InitDB(); err != nil {
		fmt.Println(colorz.Red+"[error][db.InitDB]", err, colorz.Reset)
		return
	}

	client, err := httpclient.New()
	if err != nil {
		fmt.Println(colorz.Red+"[error][httpclient.New]", err, colorz.Reset)
		return
	}

	data := cve.ReadDataset(f)
	filtered := make([]<-chan cve.Element, *workers)
	for i := range *workers {
		filtered[i] = cve.FilterEmptyURLS(data)
	}

	filteredData := cve.FanIn(filtered...)

	cve.FetchAndStore(db, client.HTTPClient, filteredData, *workers)
	renderer.RenderRequiredPages(db, *workers)
}
