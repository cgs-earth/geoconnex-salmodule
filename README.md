# geoconnex-salmodule

A [SAL module](https://github.com/cgs-earth/sal) that harvests JSON-LD straight out of the
[Geoconnex STAC catalog](https://storage.googleapis.com/harvest-geoconnex-us/stac/catalog.json).

Every Geoconnex sitemap is published there as a GeoParquet export with a `jsonld` column holding
the exact JSON-LD document the Geoconnex crawler harvested for each feature. This module reads
that column over HTTP range requests and writes it to stdout as newline delimited JSON, one
feature per line. It never downloads a whole parquet file: it fetches only the footer and the
`jsonld` column's chunks, skipping geometry, bbox, and every other column, and it starts writing
output before a file has finished streaming in. This matters because some sitemaps' GeoParquet
exports are multiple gigabytes.

## Ontology

One task class, `JsonLdHarvester`, a `salmodule:NodeProducer`. It takes two optional,
repeatable configuration properties, named after the `sitemap_id` values in
[geoconnex.us/sitemap.xml](https://geoconnex.us/sitemap.xml) (e.g. `ref:states`,
`usgs:monitoring_locations`):

- `includeSitemaps` — a whitelist. When set, only these sitemaps are harvested and
  `excludeSitemaps` is ignored.
- `excludeSitemaps` — a blacklist. When `includeSitemaps` is not set, every sitemap in the
  catalog is harvested except these.

With neither set, every sitemap in the catalog is harvested.

```turtle
@prefix salmodule: <https://w3id.org/sal/cgs-earth/sal/ontology/salmodule#> .
@prefix geoconnex: <salmodule://github.com/cgs-earth/geoconnex-salmodule/> .

<HarvestGeoconnex> a geoconnex:JsonLdHarvester ;
    geoconnex:excludeSitemaps "wwdh:usbr:rise", "wwdh:noaa_rfc" .
```

## Local testing

```sh
docker build -t geoconnex-salmodule .
docker run --rm geoconnex-salmodule salmodule ontology | jq .
docker run --rm -e SALMODULE_TASK_INSTANCE='{"@id":"https://example.org/x","@type":"JsonLdHarvester","includeSitemaps":["ref:states"]}' \
  geoconnex-salmodule salmodule run | head
```

Progress and per-sitemap counts are logged to stderr; stdout carries only the harvested JSON-LD.

## How it streams

`rangereader.go` implements an `io.ReaderAt` over plain HTTP `Range` requests, with a small
read-ahead cache so the many small, mostly sequential reads parquet-go issues inside one column
chunk collapse into a handful of large requests rather than one round trip per page.
`harvest.go` opens the remote parquet file against that reader, resolves the `jsonld` column's
index once, and reads only that column's pages across every row group, compacting and writing
each value as its own line as soon as it is decoded.
