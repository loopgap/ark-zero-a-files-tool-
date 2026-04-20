package storage

import (
	"context"
	"testing"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
)

type fakeIndexReader struct {
	closed bool
}

func (f *fakeIndexReader) Search(context.Context, bluge.SearchRequest) (search.DocumentMatchIterator, error) {
	return fakeDocumentMatchIterator{}, nil
}

func (f *fakeIndexReader) Close() error {
	f.closed = true
	return nil
}

type fakeDocumentMatchIterator struct{}

func (fakeDocumentMatchIterator) Next() (*search.DocumentMatch, error) { return nil, nil }
func (fakeDocumentMatchIterator) Aggregations() *search.Bucket         { return nil }

func TestSearchDocumentsClosesReader(t *testing.T) {
	reader := &fakeIndexReader{}
	storage := &IndexStorage{
		openReader: func() (indexReader, error) {
			return reader, nil
		},
	}

	if _, err := storage.SearchDocuments(context.Background(), "", 10); err != nil {
		t.Fatalf("SearchDocuments returned error: %v", err)
	}
	if !reader.closed {
		t.Fatal("SearchDocuments did not close the reader")
	}
}
