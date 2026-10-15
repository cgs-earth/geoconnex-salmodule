package main

import (
	"fmt"
	"io"
	"net/http"
)

// runTask harvests the jsonld column of every configured sitemap's
// GeoParquet asset and writes it to stdout as newline delimited JSON.
func runTask(stdout, stderr io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := &http.Client{}

	sources, err := listSitemapSources(client, defaultCatalogURL)
	if err != nil {
		return fmt.Errorf("listing sitemaps from the Geoconnex STAC catalog: %w", err)
	}

	sources = filterSitemaps(sources, cfg)
	if len(sources) == 0 {
		return fmt.Errorf("no sitemaps matched the configured includeSitemaps/excludeSitemaps filters")
	}

	out := bufferedStdout(stdout)
	var totalRows int64
	for _, src := range sources {
		fmt.Fprintf(stderr, "harvesting %s ...\n", src.SitemapID)
		rows, err := streamJSONLD(client, out, src.ParquetURL)
		if err != nil {
			out.Flush()
			return fmt.Errorf("harvesting sitemap %s: %w", src.SitemapID, err)
		}
		totalRows += rows
		fmt.Fprintf(stderr, "harvested %d features from %s\n", rows, src.SitemapID)

		if err := out.Flush(); err != nil {
			return err
		}
	}

	if err := out.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "done: %d features from %d sitemaps\n", totalRows, len(sources))
	return nil
}
