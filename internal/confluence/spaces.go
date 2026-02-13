package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListSpaces returns all accessible spaces.
func (c *Client) ListSpaces(limit int) ([]Space, error) {
	if c.IsCloud() {
		return c.listSpacesV2(limit)
	}
	return c.listSpacesV1(limit)
}

func (c *Client) listSpacesV2(limit int) ([]Space, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	data, err := c.getV2("spaces", q)
	if err != nil {
		return nil, err
	}

	var page CursorPage[Space]
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing spaces: %w", err)
	}
	return page.Results, nil
}

func (c *Client) listSpacesV1(limit int) ([]Space, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	data, err := c.getV1("space", q)
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []V1Space `json:"results"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing v1 spaces: %w", err)
	}

	spaces := make([]Space, len(result.Results))
	for i, s := range result.Results {
		spaces[i] = Space{
			ID:   strconv.Itoa(s.ID),
			Key:  s.Key,
			Name: s.Name,
			Type: s.Type,
		}
	}
	return spaces, nil
}

// GetSpace retrieves a single space by key.
func (c *Client) GetSpace(spaceKey string) (*Space, error) {
	if c.IsCloud() {
		id, err := c.ResolveSpaceKey(spaceKey)
		if err != nil {
			return nil, err
		}
		data, err := c.getV2("spaces/"+id, nil)
		if err != nil {
			return nil, err
		}
		var space Space
		if err := json.Unmarshal(data, &space); err != nil {
			return nil, fmt.Errorf("parsing space: %w", err)
		}
		return &space, nil
	}

	data, err := c.getV1("space/"+spaceKey, nil)
	if err != nil {
		return nil, err
	}
	var v1 V1Space
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 space: %w", err)
	}
	return &Space{
		ID:   strconv.Itoa(v1.ID),
		Key:  v1.Key,
		Name: v1.Name,
		Type: v1.Type,
	}, nil
}
