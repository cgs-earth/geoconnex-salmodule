package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/parquet-go/parquet-go"
)

const jsonldColumnName = "jsonld"

// streamJSONLD opens one sitemap's GeoParquet asset over HTTP range
// requests and writes the jsonld value of every row to w as one compact
// JSON line, skipping every other column. It never buffers the file, or
// even a whole row group's worth of other columns, in memory.
func streamJSONLD(client *http.Client, w io.Writer, parquetURL string) (int64, error) {
	reader, err := newHTTPRangeReader(client, parquetURL)
	if err != nil {
		return 0, err
	}

	pf, err := parquet.OpenFile(reader, reader.size)
	if err != nil {
		return 0, fmt.Errorf("opening %s as parquet: %w", parquetURL, err)
	}

	colIndex := -1
	for i, f := range pf.Schema().Fields() {
		if f.Name() == jsonldColumnName {
			colIndex = i
			break
		}
	}
	if colIndex == -1 {
		return 0, fmt.Errorf("%s has no %q column", parquetURL, jsonldColumnName)
	}

	var rowCount int64
	var compacted bytes.Buffer
	values := make([]parquet.Value, 256)

	for _, rg := range pf.RowGroups() {
		pages := rg.ColumnChunks()[colIndex].Pages()
		readErr := func() error {
			defer pages.Close()
			for {
				page, err := pages.ReadPage()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return fmt.Errorf("reading %s: %w", parquetURL, err)
				}

				pageErr := func() error {
					defer parquet.Release(page)
					vr := page.Values()
					for {
						n, err := vr.ReadValues(values)
						for i := 0; i < n; i++ {
							v := values[i]
							if v.IsNull() {
								continue
							}
							compacted.Reset()
							if jsonErr := json.Compact(&compacted, v.ByteArray()); jsonErr != nil {
								return fmt.Errorf("%s: row with malformed jsonld: %w", parquetURL, jsonErr)
							}
							if _, writeErr := w.Write(compacted.Bytes()); writeErr != nil {
								return writeErr
							}
							if _, writeErr := w.Write([]byte("\n")); writeErr != nil {
								return writeErr
							}
							rowCount++
						}
						if err == io.EOF {
							return nil
						}
						if err != nil {
							return fmt.Errorf("reading %s: %w", parquetURL, err)
						}
					}
				}()
				if pageErr != nil {
					return pageErr
				}
			}
		}()
		if readErr != nil {
			return rowCount, readErr
		}
	}

	return rowCount, nil
}

// bufferedStdout returns a writer that batches small line writes into
// fewer underlying syscalls; callers must Flush when done.
func bufferedStdout(w io.Writer) *bufio.Writer {
	return bufio.NewWriterSize(w, 1<<20)
}
