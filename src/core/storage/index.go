package storage

import (
	"context"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
)

type IndexStorage struct {
	writer *bluge.Writer
}

func NewIndexStorage(writer *bluge.Writer) *IndexStorage {
	return &IndexStorage{writer: writer}
}

func (s *IndexStorage) UpdateBatch(docs []bluge.Document) error {
	batch := bluge.NewBatch()
	for _, doc := range docs {
		batch.Update(doc.ID(), doc)
	}
	return s.writer.Batch(batch)
}

func (s *IndexStorage) DeleteBatch(ids []bluge.Identifier) error {
	batch := bluge.NewBatch()
	for _, id := range ids {
		batch.Delete(id)
	}
	return s.writer.Batch(batch)
}

func (s *IndexStorage) Search(ctx context.Context, query bluge.Query) (search.DocumentMatchIterator, error) {
	reader, err := s.writer.Reader()
	if err != nil {
		return nil, err
	}
	return reader.Search(ctx, bluge.NewTopNSearch(10, query))
}

func (s *IndexStorage) Close() error {
	return s.writer.Close()
}
