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

func (s *IndexStorage) SearchDocuments(ctx context.Context, keyword string, limit int) (search.DocumentMatchIterator, error) {
	if limit <= 0 {
		limit = 200
	}

	var query bluge.Query
	if keyword == "" {
		query = bluge.NewMatchAllQuery()
	} else {
		boolQuery := bluge.NewBooleanQuery()
		boolQuery.SetMinShould(1)
		boolQuery.AddShould(bluge.NewMatchQuery(keyword).SetField("name"))
		boolQuery.AddShould(bluge.NewMatchQuery(keyword).SetField("path_text"))
		boolQuery.AddShould(bluge.NewMatchQuery(keyword).SetField("body"))
		query = boolQuery
	}

	reader, err := s.writer.Reader()
	if err != nil {
		return nil, err
	}
	return reader.Search(ctx, bluge.NewTopNSearch(limit, query))
}

func (s *IndexStorage) AdvancedSearch(ctx context.Context, keyword string, category string, tag string) (search.DocumentMatchIterator, error) {
	return s.SearchDocuments(ctx, keyword, 200)
}

func (s *IndexStorage) Close() error {
	return s.writer.Close()
}
