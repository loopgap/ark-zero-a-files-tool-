package storage

import (
	"context"
	"strings"
	"unicode"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
)

type SearchDocument struct {
	Path             string
	Name             string
	RootID           string
	Extension        string
	VirtualFolderIDs []string
}

type SearchQuery struct {
	Keyword         string
	Limit           int
	Fields          []string
	RootID          string
	VirtualFolderID string
	Extension       string
}

type indexReader interface {
	Search(ctx context.Context, req bluge.SearchRequest) (search.DocumentMatchIterator, error)
	Close() error
}

type IndexStorage struct {
	writer     *bluge.Writer
	openReader func() (indexReader, error)
}

func NewIndexStorage(writer *bluge.Writer) *IndexStorage {
	return &IndexStorage{
		writer: writer,
		openReader: func() (indexReader, error) {
			return writer.Reader()
		},
	}
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

func (s *IndexStorage) SearchDocuments(ctx context.Context, keyword string, limit int) ([]SearchDocument, error) {
	return s.SearchDocumentsWithQuery(ctx, SearchQuery{Keyword: keyword, Limit: limit})
}

func (s *IndexStorage) SearchDocumentsWithQuery(ctx context.Context, query SearchQuery) ([]SearchDocument, error) {
	if query.Limit <= 0 {
		query.Limit = 200
	}

	reader, err := s.openReader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	iter, err := reader.Search(ctx, buildSearchRequest(query))
	if err != nil {
		return nil, err
	}

	results := make([]SearchDocument, 0, query.Limit)
	match, err := iter.Next()
	for err == nil && match != nil {
		item := SearchDocument{VirtualFolderIDs: []string{}}
		_ = match.VisitStoredFields(func(field string, value []byte) bool {
			switch field {
			case "_id", "path":
				item.Path = string(value)
			case "name":
				item.Name = string(value)
			case "root_id":
				item.RootID = string(value)
			case "ext":
				item.Extension = string(value)
			case "virtual_folder":
				item.VirtualFolderIDs = append(item.VirtualFolderIDs, string(value))
			}
			return true
		})
		if item.Path != "" {
			results = append(results, item)
		}
		match, err = iter.Next()
	}
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *IndexStorage) Close() error {
	return s.writer.Close()
}

func buildSearchRequest(query SearchQuery) bluge.SearchRequest {
	var baseQuery bluge.Query
	keyword := strings.TrimSpace(query.Keyword)
	fields := normalizeSearchFields(query.Fields)
	if keyword == "" {
		baseQuery = bluge.NewMatchAllQuery()
	} else {
		baseQuery = buildKeywordSearchQuery(keyword, fields)
	}

	searchQuery := bluge.NewBooleanQuery()
	searchQuery.AddMust(baseQuery)
	if query.RootID != "" {
		searchQuery.AddMust(bluge.NewTermQuery(query.RootID).SetField("root_id"))
	}
	if query.VirtualFolderID != "" {
		searchQuery.AddMust(bluge.NewTermQuery(query.VirtualFolderID).SetField("virtual_folder"))
	}
	if query.Extension != "" {
		searchQuery.AddMust(bluge.NewTermQuery(query.Extension).SetField("ext"))
	}

	return bluge.NewTopNSearch(query.Limit, searchQuery)
}

func buildKeywordSearchQuery(keyword string, fields []string) bluge.Query {
	if len(fields) == 1 {
		return buildFieldSearchQuery(fields[0], keyword)
	}
	boolQuery := bluge.NewBooleanQuery()
	boolQuery.SetMinShould(1)
	for _, field := range fields {
		boolQuery.AddShould(buildFieldSearchQuery(field, keyword))
	}
	return boolQuery
}

func buildFieldSearchQuery(field string, keyword string) bluge.Query {
	boolQuery := bluge.NewBooleanQuery()
	boolQuery.SetMinShould(1)
	seen := map[string]bool{}
	terms := append([]string{strings.TrimSpace(keyword)}, searchKeywordTokens(keyword)...)
	for _, term := range terms {
		normalized := strings.TrimSpace(term)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		boolQuery.AddShould(bluge.NewMatchQuery(normalized).SetField(field))
	}
	return boolQuery
}

func searchKeywordTokens(keyword string) []string {
	rawTokens := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(keyword)), splitSearchKeywordToken)
	if len(rawTokens) == 0 {
		return nil
	}
	out := make([]string, 0, len(rawTokens))
	seen := map[string]bool{}
	for _, token := range rawTokens {
		token = strings.TrimSpace(token)
		if len(token) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func splitSearchKeywordToken(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsNumber(r)
}

func normalizeSearchFields(fields []string) []string {
	if len(fields) == 0 {
		return []string{"name", "path_text", "body"}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field))
		switch normalized {
		case "name", "path_text", "body":
			if !seen[normalized] {
				seen[normalized] = true
				out = append(out, normalized)
			}
		}
	}
	if len(out) == 0 {
		return []string{"name", "path_text", "body"}
	}
	return out
}
