package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
)

// defaultCatalogURL is the root of the Geoconnex STAC catalog. Every
// GeoParquet export it lists is a child collection's "data" asset, keyed by
// the dataset's Geoconnex sitemap_id.
const defaultCatalogURL = "https://storage.googleapis.com/harvest-geoconnex-us/stac/catalog.json"

type stacLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type stacCatalog struct {
	Links []stacLink `json:"links"`
}

type stacCollection struct {
	ID     string `json:"id"`
	Assets struct {
		Data struct {
			Href string `json:"href"`
		} `json:"data"`
	} `json:"assets"`
}

// sitemapSource is one sitemap's GeoParquet asset, keyed by its
// Geoconnex sitemap_id (e.g. "ref:states").
type sitemapSource struct {
	SitemapID  string
	ParquetURL string
}

func fetchJSON(client *http.Client, rawURL string, out any) error {
	resp, err := client.Get(rawURL)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: unexpected status %s", rawURL, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s: %w", rawURL, err)
	}
	return nil
}

// listSitemapSources fetches the STAC catalog's child collections and
// returns every sitemap's GeoParquet asset, sorted by sitemap_id for a
// deterministic harvest order.
func listSitemapSources(client *http.Client, catalogURL string) ([]sitemapSource, error) {
	base, err := url.Parse(catalogURL)
	if err != nil {
		return nil, fmt.Errorf("parsing catalog URL %s: %w", catalogURL, err)
	}

	var catalog stacCatalog
	if err := fetchJSON(client, catalogURL, &catalog); err != nil {
		return nil, err
	}

	var sources []sitemapSource
	for _, link := range catalog.Links {
		if link.Rel != "child" {
			continue
		}
		childURL, err := base.Parse(link.Href)
		if err != nil {
			return nil, fmt.Errorf("resolving collection URL %s: %w", link.Href, err)
		}

		var collection stacCollection
		if err := fetchJSON(client, childURL.String(), &collection); err != nil {
			return nil, err
		}
		if collection.ID == "" || collection.Assets.Data.Href == "" {
			continue
		}
		sources = append(sources, sitemapSource{
			SitemapID:  collection.ID,
			ParquetURL: collection.Assets.Data.Href,
		})
	}

	sort.Slice(sources, func(i, j int) bool { return sources[i].SitemapID < sources[j].SitemapID })
	return sources, nil
}

// filterSitemaps applies the harvester's include/exclude configuration.
// A non-empty include list is a whitelist and takes precedence over
// exclude. Otherwise every sitemap not named in exclude is kept.
func filterSitemaps(sources []sitemapSource, cfg config) []sitemapSource {
	if len(cfg.IncludeSitemaps) > 0 {
		include := toSet(cfg.IncludeSitemaps)
		var kept []sitemapSource
		for _, s := range sources {
			if include[s.SitemapID] {
				kept = append(kept, s)
			}
		}
		return kept
	}

	exclude := toSet(cfg.ExcludeSitemaps)
	var kept []sitemapSource
	for _, s := range sources {
		if !exclude[s.SitemapID] {
			kept = append(kept, s)
		}
	}
	return kept
}

func toSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}
