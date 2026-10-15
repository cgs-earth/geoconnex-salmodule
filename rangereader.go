package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

// readAheadBytes is how much a single range request fetches past the bytes
// parquet-go actually asked for. Parquet column data within one chunk is
// read in small, mostly-increasing offsets (page headers, then page data),
// so a modest read-ahead collapses what would otherwise be one HTTP round
// trip per page into a handful of requests per column chunk.
const readAheadBytes = 4 << 20 // 4 MiB

// httpRangeReader is an io.ReaderAt over a remote file, fetched lazily via
// HTTP range requests. It never downloads the file up front: parquet-go
// only calls ReadAt for the footer and for the specific column chunks a
// read touches, so unselected columns (geometry, bbox, ...) are never
// requested at all.
type httpRangeReader struct {
	url    string
	size   int64
	client *http.Client

	mu       sync.Mutex
	cacheOff int64
	cacheBuf []byte
}

func newHTTPRangeReader(client *http.Client, rawURL string) (*httpRangeReader, error) {
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HEAD %s: %w", rawURL, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HEAD %s: unexpected status %s", rawURL, resp.Status)
	}
	if resp.ContentLength < 0 {
		return nil, fmt.Errorf("HEAD %s: server did not report a content length", rawURL)
	}

	return &httpRangeReader{url: rawURL, size: resp.ContentLength, client: client}, nil
}

func (r *httpRangeReader) fetch(off, length int64) ([]byte, error) {
	end := off + length - 1
	if end >= r.size {
		end = r.size - 1
	}

	req, err := http.NewRequest(http.MethodGet, r.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, end))

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", r.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", r.url, resp.Status)
	}

	buf := make([]byte, end-off+1)
	n, err := io.ReadFull(resp.Body, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", r.url, err)
	}
	return buf[:n], nil
}

// ReadAt implements io.ReaderAt, serving requests from an in-memory
// read-ahead cache so sequential small reads collapse into a few large
// range requests.
func (r *httpRangeReader) ReadAt(p []byte, off int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	need := off + int64(len(p))
	cacheEnd := r.cacheOff + int64(len(r.cacheBuf))
	if r.cacheBuf == nil || off < r.cacheOff || need > cacheEnd {
		fetchLen := int64(len(p))
		if fetchLen < readAheadBytes {
			fetchLen = readAheadBytes
		}
		buf, err := r.fetch(off, fetchLen)
		if err != nil {
			return 0, err
		}
		r.cacheOff = off
		r.cacheBuf = buf
	}

	relOff := off - r.cacheOff
	n := copy(p, r.cacheBuf[relOff:])
	if n < len(p) {
		return n, io.ErrUnexpectedEOF
	}
	return n, nil
}
