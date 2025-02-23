/*
Package main implements command-line functionality.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/vigo/cvepreserve/internal/cve"
	"github.com/vigo/cvepreserve/internal/db/sqlite"
	"github.com/vigo/cvepreserve/internal/httpclient"
	"github.com/vigo/cvepreserve/internal/renderer"
	"github.com/vigo/cvepreserve/internal/tlog"
	"github.com/vigo/cvepreserve/internal/version"
)

const (
	defaultShowVersion  = false
	defaultDatasetFile  = "dataset.json"
	defaultLogLevel     = "info"
	defaultLogColorized = true
)

func main() {
	vrs := flag.Bool("version", defaultShowVersion, "display version information")
	logLevel := flag.String("loglevel", defaultLogLevel, "log level")
	logColorized := flag.Bool("logcolorized", defaultLogColorized, "log colorized")
	dataset := flag.String("dataset", defaultDatasetFile, "dataset json filename")
	workers := flag.Int("workers", runtime.NumCPU(), "number of concurrent workers")
	flag.Parse()

	if *vrs {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", version.Version)
		return
	}

	logger := tlog.New(*logLevel, *logColorized)

	f, err := os.Open(*dataset)
	if err != nil {
		logger.Error("dataset open", "err", err)
		return
	}

	db, err := sqlite.New(
		sqlite.WithTargetSqliteFilename("result.sqlite3"),
	)
	if err != nil {
		logger.Error("instantiate db", "err", err)
		return
	}

	defer func() {
		_ = db.DB.Close()
	}()

	if err = db.InitDB(); err != nil {
		logger.Error("init db", "err", err)
		return
	}

	client, err := httpclient.New()
	if err != nil {
		logger.Error("instantiate http client", "err", err)
		return
	}

	data := cve.ReadDataset(f, logger)
	filtered := make([]<-chan cve.Element, *workers)
	for i := range *workers {
		filtered[i] = cve.FilterEmptyURLS(data)
	}

	filteredData := cve.FanIn(filtered...)

	cve.FetchAndStore(db, client.HTTPClient, filteredData, *workers, logger)
	renderer.RenderRequiredPages(db, *workers, logger)
}
