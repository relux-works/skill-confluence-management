package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// SearchCQL performs a CQL search (always v1 — no v2 search endpoint exists).
func (c *Client) SearchCQL(cql string, limit int) (*SearchResult, error) {
	q := url.Values{"cql": {cql}}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	// Use /rest/api/search for rich results (includes excerpts).
	data, err := c.getV1("search", q)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}
	return &result, nil
}

// SearchContentCQL performs a content-only CQL search (v1, simpler response).
func (c *Client) SearchContentCQL(cql string, limit int, expand string) ([]V1Content, error) {
	q := url.Values{"cql": {cql}}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if expand != "" {
		q.Set("expand", expand)
	}

	data, err := c.getV1("content/search", q)
	if err != nil {
		return nil, err
	}

	var result V1PageResults
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing content search: %w", err)
	}
	return result.Results, nil
}
