package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"short-video-platform/video-service/internal/model"
)

const defaultIndex = "svp_videos"

type Client struct {
	es    *elasticsearch.Client
	index string
}

type videoDoc struct {
	VideoID     string   `json:"video_id"`
	UserID      string   `json:"user_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	CreatedAt   int64    `json:"created_at"`
	Tags        []string `json:"tags,omitempty"`
}

type searchHit struct {
	Source videoDoc `json:"_source"`
}

type searchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []searchHit `json:"hits"`
	} `json:"hits"`
}

func NewFromEnv() (*Client, error) {
	url := strings.TrimSpace(os.Getenv("ELASTICSEARCH_URL"))
	if url == "" {
		return nil, nil
	}
	index := strings.TrimSpace(os.Getenv("ELASTICSEARCH_INDEX"))
	if index == "" {
		index = defaultIndex
	}
	cfg := elasticsearch.Config{
		Addresses: []string{url},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	c := &Client{es: es, index: index}
	if err := c.ensureIndex(context.Background()); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ensureIndex(ctx context.Context) error {
	res, err := c.es.Indices.Exists([]string{c.index}, c.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return nil
	}

	body := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"video_id":    map[string]string{"type": "keyword"},
				"user_id":     map[string]string{"type": "keyword"},
				"title":       map[string]string{"type": "text"},
				"description": map[string]string{"type": "text"},
				"status":      map[string]string{"type": "keyword"},
				"created_at":  map[string]string{"type": "long"},
				"tags":        map[string]string{"type": "keyword"},
			},
		},
	}
	raw, _ := json.Marshal(body)
	createRes, err := c.es.Indices.Create(
		c.index,
		c.es.Indices.Create.WithContext(ctx),
		c.es.Indices.Create.WithBody(bytes.NewReader(raw)),
	)
	if err != nil {
		return err
	}
	defer createRes.Body.Close()
	if createRes.IsError() && createRes.StatusCode != 400 {
		return readEsError(createRes)
	}
	return nil
}

func (c *Client) Index(ctx context.Context, v *model.Video) error {
	if c == nil || v == nil {
		return nil
	}
	doc := videoDoc{
		VideoID:     v.ID,
		UserID:      v.UserID,
		Title:       v.Title,
		Description: v.Description,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		Tags:        v.Tags,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := c.es.Index(
		c.index,
		bytes.NewReader(raw),
		c.es.Index.WithDocumentID(v.ID),
		c.es.Index.WithContext(ctx),
		c.es.Index.WithRefresh("false"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return readEsError(res)
	}
	return nil
}

func (c *Client) ReindexAll(ctx context.Context, videos []model.Video) error {
	if c == nil {
		return nil
	}
	for i := range videos {
		if err := c.Index(ctx, &videos[i]); err != nil {
			return fmt.Errorf("index %s: %w", videos[i].ID, err)
		}
	}
	return nil
}

func (c *Client) Search(ctx context.Context, keyword string, page, pageSize int) ([]string, int, error) {
	if c == nil {
		return nil, 0, fmt.Errorf("elasticsearch disabled")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	from := (page - 1) * pageSize

	query := map[string]any{
		"from": from,
		"size": pageSize,
		"sort": []any{
			map[string]any{"created_at": map[string]string{"order": "desc"}},
		},
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"title^3", "description"},
			},
		},
	}
	raw, _ := json.Marshal(query)
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.index),
		c.es.Search.WithBody(bytes.NewReader(raw)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, 0, readEsError(res)
	}

	var parsed searchResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, err
	}

	ids := make([]string, 0, len(parsed.Hits.Hits))
	for _, hit := range parsed.Hits.Hits {
		if hit.Source.VideoID != "" {
			ids = append(ids, hit.Source.VideoID)
		}
	}
	return ids, parsed.Hits.Total.Value, nil
}

func (c *Client) Delete(ctx context.Context, videoID string) error {
	if c == nil || videoID == "" {
		return nil
	}
	res, err := c.es.Delete(c.index, videoID, c.es.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		return readEsError(res)
	}
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("elasticsearch disabled")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return readEsError(res)
	}
	return nil
}

func readEsError(res *esapi.Response) error {
	body, _ := io.ReadAll(res.Body)
	return fmt.Errorf("elasticsearch %s: %s", res.Status(), strings.TrimSpace(string(body)))
}
