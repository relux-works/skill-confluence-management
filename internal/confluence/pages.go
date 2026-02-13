package confluence

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// GetPage retrieves a page by ID (v2 Cloud, v1 Server/DC).
func (c *Client) GetPage(pageID string, includeBody bool) (*Page, error) {
	if c.IsCloud() {
		return c.getPageV2(pageID, includeBody)
	}
	return c.getPageV1(pageID, includeBody)
}

func (c *Client) getPageV2(pageID string, includeBody bool) (*Page, error) {
	q := url.Values{}
	if includeBody {
		q.Set("body-format", "storage")
	}

	data, err := c.getV2("pages/"+pageID, q)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing page: %w", err)
	}
	return &page, nil
}

func (c *Client) getPageV1(pageID string, includeBody bool) (*Page, error) {
	q := url.Values{}
	expand := "version,space,ancestors,metadata.labels"
	if includeBody {
		expand += ",body.storage"
	}
	q.Set("expand", expand)

	data, err := c.getV1("content/"+pageID, q)
	if err != nil {
		return nil, err
	}

	var v1 V1Content
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 page: %w", err)
	}
	return v1ToPage(&v1), nil
}

// ListPages lists pages in a space (v2 Cloud, v1 Server/DC).
func (c *Client) ListPages(spaceKey string, title string, limit int) ([]Page, error) {
	if c.IsCloud() {
		return c.listPagesV2(spaceKey, title, limit)
	}
	return c.listPagesV1(spaceKey, title, limit)
}

func (c *Client) listPagesV2(spaceKey string, title string, limit int) ([]Page, error) {
	spaceID, err := c.ResolveSpaceKey(spaceKey)
	if err != nil {
		return nil, err
	}

	q := url.Values{"space-id": {spaceID}}
	if title != "" {
		q.Set("title", title)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	data, err := c.getV2("pages", q)
	if err != nil {
		return nil, err
	}

	var page CursorPage[Page]
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing pages: %w", err)
	}
	return page.Results, nil
}

func (c *Client) listPagesV1(spaceKey string, title string, limit int) ([]Page, error) {
	q := url.Values{
		"type":     {"page"},
		"spaceKey": {spaceKey},
		"expand":   {"version,space"},
	}
	if title != "" {
		q.Set("title", title)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	data, err := c.getV1("content", q)
	if err != nil {
		return nil, err
	}

	var result V1PageResults
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing v1 pages: %w", err)
	}

	pages := make([]Page, len(result.Results))
	for i := range result.Results {
		pages[i] = *v1ToPage(&result.Results[i])
	}
	return pages, nil
}

// GetChildren retrieves direct children of a page (v2 Cloud, v1 Server/DC).
func (c *Client) GetChildren(pageID string, limit int) ([]Page, error) {
	if c.IsCloud() {
		return c.getChildrenV2(pageID, limit)
	}
	return c.getChildrenV1(pageID, limit)
}

func (c *Client) getChildrenV2(pageID string, limit int) ([]Page, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	data, err := c.getV2("pages/"+pageID+"/children", q)
	if err != nil {
		return nil, err
	}

	var page CursorPage[Page]
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing children: %w", err)
	}
	return page.Results, nil
}

func (c *Client) getChildrenV1(pageID string, limit int) ([]Page, error) {
	q := url.Values{"expand": {"version"}}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	data, err := c.getV1("content/"+pageID+"/child/page", q)
	if err != nil {
		return nil, err
	}

	var result V1PageResults
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing v1 children: %w", err)
	}

	pages := make([]Page, len(result.Results))
	for i := range result.Results {
		pages[i] = *v1ToPage(&result.Results[i])
	}
	return pages, nil
}

// GetAncestors retrieves the breadcrumb chain for a page (v2 Cloud only, v1 via expand).
func (c *Client) GetAncestors(pageID string) ([]Ancestor, error) {
	if c.IsCloud() {
		data, err := c.getV2("pages/"+pageID+"/ancestors", nil)
		if err != nil {
			return nil, err
		}
		var result CursorPage[Ancestor]
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parsing ancestors: %w", err)
		}
		return result.Results, nil
	}

	// V1: get page with ancestors expanded.
	q := url.Values{"expand": {"ancestors"}}
	data, err := c.getV1("content/"+pageID, q)
	if err != nil {
		return nil, err
	}
	var v1 V1Content
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 ancestors: %w", err)
	}
	ancestors := make([]Ancestor, len(v1.Ancestors))
	for i, a := range v1.Ancestors {
		ancestors[i] = Ancestor{ID: a.ID, Title: a.Title}
	}
	return ancestors, nil
}

// CreatePage creates a new page (v2 Cloud, v1 Server/DC).
func (c *Client) CreatePage(spaceKey, title, body, parentID string) (*Page, error) {
	if c.IsCloud() {
		return c.createPageV2(spaceKey, title, body, parentID)
	}
	return c.createPageV1(spaceKey, title, body, parentID)
}

func (c *Client) createPageV2(spaceKey, title, body, parentID string) (*Page, error) {
	spaceID, err := c.ResolveSpaceKey(spaceKey)
	if err != nil {
		return nil, err
	}

	req := CreatePageRequest{
		SpaceID:  spaceID,
		Status:   "current",
		Title:    title,
		ParentID: parentID,
	}
	if body != "" {
		req.Body = &CreatePageBody{
			Representation: "storage",
			Value:          body,
		}
	}

	data, err := c.postV2("pages", req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing created page: %w", err)
	}
	return &page, nil
}

func (c *Client) createPageV1(spaceKey, title, body, parentID string) (*Page, error) {
	v1Req := map[string]interface{}{
		"type":  "page",
		"title": title,
		"space": map[string]string{"key": spaceKey},
	}
	if body != "" {
		v1Req["body"] = map[string]interface{}{
			"storage": map[string]string{
				"value":          body,
				"representation": "storage",
			},
		}
	}
	if parentID != "" {
		v1Req["ancestors"] = []map[string]string{{"id": parentID}}
	}

	data, err := c.postV1("content", v1Req)
	if err != nil {
		return nil, err
	}

	var v1 V1Content
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 created page: %w", err)
	}
	return v1ToPage(&v1), nil
}

// UpdatePage updates an existing page (v2 Cloud, v1 Server/DC).
// Automatically handles version increment.
func (c *Client) UpdatePage(pageID, title, body, message string) (*Page, error) {
	// First, get current version.
	current, err := c.GetPage(pageID, false)
	if err != nil {
		return nil, fmt.Errorf("reading current version: %w", err)
	}

	currentVersion := 0
	if current.Version != nil {
		currentVersion = current.Version.Number
	}

	if title == "" {
		title = current.Title
	}

	if c.IsCloud() {
		return c.updatePageV2(pageID, title, body, message, currentVersion+1)
	}
	return c.updatePageV1(pageID, title, body, message, currentVersion+1)
}

func (c *Client) updatePageV2(pageID, title, body, message string, versionNumber int) (*Page, error) {
	req := UpdatePageRequest{
		ID:     pageID,
		Status: "current",
		Title:  title,
		Version: &VersionUpdate{
			Number:  versionNumber,
			Message: message,
		},
	}
	if body != "" {
		req.Body = &CreatePageBody{
			Representation: "storage",
			Value:          body,
		}
	}

	data, err := c.putV2("pages/"+pageID, req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing updated page: %w", err)
	}
	return &page, nil
}

func (c *Client) updatePageV1(pageID, title, body, message string, versionNumber int) (*Page, error) {
	v1Req := map[string]interface{}{
		"type":  "page",
		"title": title,
		"version": map[string]interface{}{
			"number":  versionNumber,
			"message": message,
		},
	}
	if body != "" {
		v1Req["body"] = map[string]interface{}{
			"storage": map[string]string{
				"value":          body,
				"representation": "storage",
			},
		}
	}

	fullURL := c.v1URL("content/" + pageID)
	data, err := c.request(http.MethodPut, fullURL, nil, v1Req)
	if err != nil {
		return nil, err
	}

	var v1 V1Content
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 updated page: %w", err)
	}
	return v1ToPage(&v1), nil
}

// DeletePage trashes a page (v2 Cloud, v1 Server/DC).
func (c *Client) DeletePage(pageID string) error {
	if c.IsCloud() {
		_, err := c.deleteV2("pages/" + pageID)
		return err
	}
	fullURL := c.v1URL("content/" + pageID)
	_, err := c.request(http.MethodDelete, fullURL, nil, nil)
	return err
}

// postV1 performs a POST request to the v1 API.
func (c *Client) postV1(path string, body interface{}) ([]byte, error) {
	fullURL := c.v1URL(path)
	return c.request(http.MethodPost, fullURL, nil, body)
}

// --- V1 to V2 type conversion ---

func v1ToPage(v1 *V1Content) *Page {
	p := &Page{
		ID:     v1.ID,
		Status: v1.Status,
		Title:  v1.Title,
	}

	if v1.Space != nil {
		p.SpaceID = strconv.Itoa(v1.Space.ID)
	}

	if v1.Version != nil {
		p.Version = &Version{
			Number:  v1.Version.Number,
			Message: v1.Version.Message,
		}
		if v1.Version.When != "" {
			p.Version.CreatedAt = v1.Version.When
		}
		if v1.Version.By != nil {
			p.Version.AuthorID = v1.Version.By.AccountID
			if p.Version.AuthorID == "" {
				p.Version.AuthorID = v1.Version.By.Username
			}
		}
	}

	if v1.Body != nil && v1.Body.Storage != nil {
		p.Body = &PageBody{
			Storage: &BodyRepresentation{
				Value:          v1.Body.Storage.Value,
				Representation: v1.Body.Storage.Representation,
			},
		}
	}

	if v1.Metadata != nil && v1.Metadata.Labels != nil {
		p.Labels = &LabelArray{Results: v1.Metadata.Labels.Results}
	}

	return p
}
