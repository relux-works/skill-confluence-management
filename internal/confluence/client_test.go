package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	ts := httptest.NewServer(handler)
	client, _ := NewClient(Config{
		BaseURL:      ts.URL,
		Email:        "test@test.com",
		Token:        "tok",
		InstanceType: InstanceCloud,
		AuthType:     AuthBasic,
	})
	client.SetHTTPClient(ts.Client())
	return ts, client
}

func newTestServerV1(handler http.HandlerFunc) (*httptest.Server, *Client) {
	ts := httptest.NewServer(handler)
	client, _ := NewClient(Config{
		BaseURL:      ts.URL,
		Token:        "pat",
		InstanceType: InstanceServer,
		AuthType:     AuthBearer,
	})
	client.SetHTTPClient(ts.Client())
	return ts, client
}

func TestNewClient_BasicAuth(t *testing.T) {
	c, err := NewClient(Config{
		BaseURL: "https://x.atlassian.net/wiki",
		Email:   "a@b.com",
		Token:   "tok",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(c.authHeader, "Basic ") {
		t.Errorf("expected Basic auth header, got %q", c.authHeader)
	}
}

func TestNewClient_BearerAuth(t *testing.T) {
	c, err := NewClient(Config{
		BaseURL:  "https://confluence.co",
		Token:    "my-pat",
		AuthType: AuthBearer,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.authHeader != "Bearer my-pat" {
		t.Errorf("expected Bearer auth, got %q", c.authHeader)
	}
}

func TestNewClient_MissingBaseURL(t *testing.T) {
	_, err := NewClient(Config{Token: "tok"})
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
}

func TestNewClient_MissingToken(t *testing.T) {
	_, err := NewClient(Config{BaseURL: "https://x.atlassian.net/wiki"})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestNewClient_BasicAuthNeedsEmail(t *testing.T) {
	_, err := NewClient(Config{
		BaseURL:  "https://x.atlassian.net/wiki",
		Token:    "tok",
		AuthType: AuthBasic,
	})
	if err == nil {
		t.Fatal("expected error for basic auth without email")
	}
}

func TestClient_IsCloud(t *testing.T) {
	c, _ := NewClient(Config{
		BaseURL:      "https://x.atlassian.net/wiki",
		Token:        "pat",
		AuthType:     AuthBearer,
		InstanceType: InstanceCloud,
	})
	if !c.IsCloud() {
		t.Error("expected IsCloud() = true")
	}

	c2, _ := NewClient(Config{
		BaseURL:      "https://confluence.co",
		Token:        "pat",
		AuthType:     AuthBearer,
		InstanceType: InstanceServer,
	})
	if c2.IsCloud() {
		t.Error("expected IsCloud() = false for server")
	}
}

func TestClient_GetPage_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/pages/12345" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Page{
			ID:    "12345",
			Title: "Test Page",
		})
	})
	defer ts.Close()

	page, err := client.GetPage("12345", false)
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if page.ID != "12345" || page.Title != "Test Page" {
		t.Errorf("unexpected page: %+v", page)
	}
}

func TestClient_GetPage_WithBody(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("body-format") != "storage" {
			t.Error("expected body-format=storage query param")
		}
		json.NewEncoder(w).Encode(Page{
			ID:    "12345",
			Title: "Test Page",
			Body: &PageBody{
				Storage: &BodyRepresentation{Value: "<p>Hello</p>"},
			},
		})
	})
	defer ts.Close()

	page, err := client.GetPage("12345", true)
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if page.Body == nil || page.Body.Storage == nil || page.Body.Storage.Value != "<p>Hello</p>" {
		t.Errorf("expected body content, got %+v", page.Body)
	}
}

func TestClient_GetPage_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/api/content/12345") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(V1Content{
			ID:    "12345",
			Title: "Server Page",
			Space: &V1Space{ID: 1, Key: "DEV"},
			Version: &V1Version{Number: 3},
		})
	})
	defer ts.Close()

	page, err := client.GetPage("12345", false)
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if page.Title != "Server Page" {
		t.Errorf("unexpected title: %s", page.Title)
	}
	if page.Version == nil || page.Version.Number != 3 {
		t.Errorf("unexpected version: %+v", page.Version)
	}
}

func TestClient_ListSpaces_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(CursorPage[Space]{
			Results: []Space{
				{ID: "1", Key: "DEV", Name: "Development"},
				{ID: "2", Key: "OPS", Name: "Operations"},
			},
		})
	})
	defer ts.Close()

	spaces, err := client.ListSpaces(0)
	if err != nil {
		t.Fatalf("ListSpaces error: %v", err)
	}
	if len(spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(spaces))
	}
	if spaces[0].Key != "DEV" {
		t.Errorf("expected DEV, got %s", spaces[0].Key)
	}
}

func TestClient_SearchCQL(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/api/search") {
			t.Errorf("search should use v1 API, got path: %s", r.URL.Path)
		}
		cql := r.URL.Query().Get("cql")
		if cql != "type=page" {
			t.Errorf("unexpected cql: %s", cql)
		}
		json.NewEncoder(w).Encode(SearchResult{
			Results: []SearchResultItem{
				{Title: "Found Page", Content: &V1Content{ID: "99"}},
			},
			Size:      1,
			TotalSize: 1,
		})
	})
	defer ts.Close()

	result, err := client.SearchCQL("type=page", 25)
	if err != nil {
		t.Fatalf("SearchCQL error: %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].Title != "Found Page" {
		t.Errorf("unexpected results: %+v", result)
	}
}

func TestClient_ListPages_Cloud(t *testing.T) {
	call := 0
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// ResolveSpaceKey call
			json.NewEncoder(w).Encode(CursorPage[Space]{
				Results: []Space{{ID: "42", Key: "DEV"}},
			})
			return
		}
		// ListPages call
		spaceID := r.URL.Query().Get("space-id")
		if spaceID != "42" {
			t.Errorf("expected space-id=42, got %s", spaceID)
		}
		json.NewEncoder(w).Encode(CursorPage[Page]{
			Results: []Page{
				{ID: "1", Title: "Page 1"},
				{ID: "2", Title: "Page 2"},
			},
		})
	})
	defer ts.Close()

	pages, err := client.ListPages("DEV", "", 0)
	if err != nil {
		t.Fatalf("ListPages error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
}

func TestClient_GetChildren_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/pages/100/children" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(CursorPage[Page]{
			Results: []Page{
				{ID: "101", Title: "Child 1"},
				{ID: "102", Title: "Child 2"},
			},
		})
	})
	defer ts.Close()

	children, err := client.GetChildren("100", 0)
	if err != nil {
		t.Fatalf("GetChildren error: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
}

func TestClient_GetAncestors_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(CursorPage[Ancestor]{
			Results: []Ancestor{
				{ID: "1", Title: "Root"},
				{ID: "10", Title: "Parent"},
			},
		})
	})
	defer ts.Close()

	ancestors, err := client.GetAncestors("100")
	if err != nil {
		t.Fatalf("GetAncestors error: %v", err)
	}
	if len(ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(ancestors))
	}
	if ancestors[0].Title != "Root" {
		t.Errorf("first ancestor = %q, want Root", ancestors[0].Title)
	}
}

func TestClient_CreatePage_Cloud(t *testing.T) {
	call := 0
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// ResolveSpaceKey
			json.NewEncoder(w).Encode(CursorPage[Space]{
				Results: []Space{{ID: "42", Key: "DEV"}},
			})
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req CreatePageRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Title != "New Page" || req.SpaceID != "42" {
			t.Errorf("unexpected request: %+v", req)
		}
		json.NewEncoder(w).Encode(Page{ID: "999", Title: "New Page"})
	})
	defer ts.Close()

	page, err := client.CreatePage("DEV", "New Page", "<p>body</p>", "")
	if err != nil {
		t.Fatalf("CreatePage error: %v", err)
	}
	if page.ID != "999" {
		t.Errorf("expected ID 999, got %s", page.ID)
	}
}

func TestClient_DeletePage_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/pages/12345" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	err := client.DeletePage("12345")
	if err != nil {
		t.Fatalf("DeletePage error: %v", err)
	}
}

func TestClient_AddLabels_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var labels AddLabelsRequest
		json.NewDecoder(r.Body).Decode(&labels)
		if len(labels) != 2 {
			t.Errorf("expected 2 labels, got %d", len(labels))
		}
		json.NewEncoder(w).Encode(LabelArray{
			Results: []Label{{Name: "a"}, {Name: "b"}},
		})
	})
	defer ts.Close()

	err := client.AddLabels("12345", []string{"a", "b"})
	if err != nil {
		t.Fatalf("AddLabels error: %v", err)
	}
}

func TestClient_GetLabels_Cloud(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(CursorPage[Label]{
			Results: []Label{
				{ID: "1", Name: "api-docs"},
				{ID: "2", Name: "v2"},
			},
		})
	})
	defer ts.Close()

	labels, err := client.GetLabels("12345")
	if err != nil {
		t.Fatalf("GetLabels error: %v", err)
	}
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
}

func TestClient_RemoveLabel_Cloud(t *testing.T) {
	call := 0
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// GetLabels
			json.NewEncoder(w).Encode(CursorPage[Label]{
				Results: []Label{{ID: "77", Name: "remove-me"}},
			})
			return
		}
		// Delete label
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/labels/77") {
			t.Errorf("expected label ID 77 in path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	err := client.RemoveLabel("12345", "remove-me")
	if err != nil {
		t.Fatalf("RemoveLabel error: %v", err)
	}
}

func TestClient_4xxNoRetry(t *testing.T) {
	callCount := 0
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	})
	defer ts.Close()

	_, err := client.GetPage("nonexistent", false)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if callCount != 1 {
		t.Errorf("4xx should not retry, but got %d calls", callCount)
	}
}

func TestClient_AuthHeaderSent(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("expected Basic auth header, got %q", auth)
		}
		json.NewEncoder(w).Encode(Page{ID: "1"})
	})
	defer ts.Close()

	client.GetPage("1", false)
}

func TestClient_ResolveSpaceKey_Caches(t *testing.T) {
	callCount := 0
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(CursorPage[Space]{
			Results: []Space{{ID: "42", Key: "DEV"}},
		})
	})
	defer ts.Close()

	id1, err := client.ResolveSpaceKey("DEV")
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	id2, err := client.ResolveSpaceKey("DEV")
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if id1 != id2 || id1 != "42" {
		t.Errorf("expected 42 both times, got %s, %s", id1, id2)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (cached), got %d", callCount)
	}
}

func TestAPIError_Format(t *testing.T) {
	e := &APIError{StatusCode: 404, Message: "page not found"}
	if e.Error() != "page not found" {
		t.Errorf("unexpected error string: %s", e.Error())
	}

	e2 := &APIError{StatusCode: 500, ErrorMessage: "server error"}
	if e2.Error() != "server error" {
		t.Errorf("unexpected error string: %s", e2.Error())
	}

	e3 := &APIError{StatusCode: 502}
	if e3.Error() != "confluence: unknown API error" {
		t.Errorf("unexpected error string: %s", e3.Error())
	}
}

func TestClient_ListSpaces_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/space" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(struct {
			Results []V1Space `json:"results"`
		}{
			Results: []V1Space{
				{ID: 1, Key: "DEV", Name: "Development"},
			},
		})
	})
	defer ts.Close()

	spaces, err := client.ListSpaces(0)
	if err != nil {
		t.Fatalf("ListSpaces error: %v", err)
	}
	if len(spaces) != 1 || spaces[0].Key != "DEV" {
		t.Errorf("unexpected spaces: %+v", spaces)
	}
}

func TestClient_ListPages_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(V1PageResults{
			Results: []V1Content{
				{ID: "1", Title: "Page 1", Space: &V1Space{ID: 10, Key: "DEV"}},
			},
		})
	})
	defer ts.Close()

	pages, err := client.ListPages("DEV", "", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(pages) != 1 || pages[0].Title != "Page 1" {
		t.Errorf("unexpected pages: %+v", pages)
	}
}

func TestClient_GetChildren_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(V1PageResults{
			Results: []V1Content{
				{ID: "10", Title: "Child"},
			},
		})
	})
	defer ts.Close()

	children, err := client.GetChildren("1", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(children) != 1 {
		t.Errorf("expected 1 child, got %d", len(children))
	}
}

func TestClient_GetAncestors_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(V1Content{
			ID: "100",
			Ancestors: []V1Content{
				{ID: "1", Title: "Root"},
			},
		})
	})
	defer ts.Close()

	ancestors, err := client.GetAncestors("100")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(ancestors) != 1 || ancestors[0].Title != "Root" {
		t.Errorf("unexpected ancestors: %+v", ancestors)
	}
}

func TestClient_CreatePage_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(V1Content{
			ID: "99", Title: "New", Space: &V1Space{Key: "DEV"},
		})
	})
	defer ts.Close()

	page, err := client.CreatePage("DEV", "New", "<p>hi</p>", "1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if page.ID != "99" {
		t.Errorf("unexpected page ID: %s", page.ID)
	}
}

func TestClient_DeletePage_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	err := client.DeletePage("123")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestClient_GetLabels_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(V1Content{
			ID: "1",
			Metadata: &V1Metadata{
				Labels: &V1LabelResults{
					Results: []Label{{Name: "tag1"}},
				},
			},
		})
	})
	defer ts.Close()

	labels, err := client.GetLabels("1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(labels) != 1 || labels[0].Name != "tag1" {
		t.Errorf("unexpected labels: %+v", labels)
	}
}

func TestClient_AddLabels_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode([]Label{{Name: "a"}})
	})
	defer ts.Close()

	err := client.AddLabels("1", []string{"a"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestClient_RemoveLabel_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	err := client.RemoveLabel("1", "tag")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestClient_SearchContentCQL(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/content/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(V1PageResults{
			Results: []V1Content{{ID: "1", Title: "Found"}},
		})
	})
	defer ts.Close()

	results, err := client.SearchContentCQL("type=page", 10, "version")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Found" {
		t.Errorf("unexpected: %+v", results)
	}
}

func TestClient_GetSpace_Cloud(t *testing.T) {
	call := 0
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// ResolveSpaceKey
			json.NewEncoder(w).Encode(CursorPage[Space]{
				Results: []Space{{ID: "42", Key: "DEV", Name: "Dev"}},
			})
			return
		}
		json.NewEncoder(w).Encode(Space{ID: "42", Key: "DEV", Name: "Development"})
	})
	defer ts.Close()

	space, err := client.GetSpace("DEV")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if space.Key != "DEV" {
		t.Errorf("unexpected space: %+v", space)
	}
}

func TestClient_GetSpace_Server(t *testing.T) {
	ts, client := newTestServerV1(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(V1Space{ID: 5, Key: "OPS", Name: "Operations"})
	})
	defer ts.Close()

	space, err := client.GetSpace("OPS")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if space.Key != "OPS" || space.Name != "Operations" {
		t.Errorf("unexpected space: %+v", space)
	}
}

func TestClient_RemoveLabel_NotFound(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(CursorPage[Label]{
			Results: []Label{{ID: "1", Name: "other"}},
		})
	})
	defer ts.Close()

	err := client.RemoveLabel("1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing label")
	}
}

func TestV1ToPage_Conversion(t *testing.T) {
	v1 := &V1Content{
		ID:     "99",
		Title:  "V1 Page",
		Status: "current",
		Space:  &V1Space{ID: 10, Key: "DEV"},
		Version: &V1Version{
			Number: 5,
			By:     &V1User{AccountID: "user-1", DisplayName: "Test"},
		},
		Body: &V1Body{
			Storage: &V1BodyContent{Value: "<p>content</p>", Representation: "storage"},
		},
		Metadata: &V1Metadata{
			Labels: &V1LabelResults{
				Results: []Label{{Name: "tag1"}, {Name: "tag2"}},
			},
		},
	}

	p := v1ToPage(v1)
	if p.ID != "99" || p.Title != "V1 Page" {
		t.Errorf("basic fields: %+v", p)
	}
	if p.SpaceID != "10" {
		t.Errorf("spaceID = %s, want 10", p.SpaceID)
	}
	if p.Version.Number != 5 || p.Version.AuthorID != "user-1" {
		t.Errorf("version = %+v", p.Version)
	}
	if p.Body == nil || p.Body.Storage.Value != "<p>content</p>" {
		t.Errorf("body = %+v", p.Body)
	}
	if p.Labels == nil || len(p.Labels.Results) != 2 {
		t.Errorf("labels = %+v", p.Labels)
	}
}
