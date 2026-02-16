package query

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/relux-works/skill-agent-facing-api/agentquery"
	"github.com/relux-works/skill-confluence-management/internal/confluence"
)

// newTestServer creates an httptest server and a connected Confluence client.
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *confluence.Client) {
	ts := httptest.NewServer(handler)
	client, _ := confluence.NewClient(confluence.Config{
		BaseURL:      ts.URL,
		Email:        "test@test.com",
		Token:        "tok",
		InstanceType: confluence.InstanceCloud,
		AuthType:     confluence.AuthBasic,
	})
	client.SetHTTPClient(ts.Client())
	return ts, client
}

// queryJSON executes a query via the schema and returns the raw JSON string.
func queryJSON(t *testing.T, schema *agentquery.Schema[*confluence.Page], input string) string {
	t.Helper()
	data, err := schema.QueryJSONWithMode(input, agentquery.HumanReadable)
	if err != nil {
		t.Fatalf("query %q failed: %v", input, err)
	}
	return string(data)
}

func TestSchema_GetByID(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.Page{
			ID:    "12345",
			Title: "Test Page",
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `get(12345){minimal}`)

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["id"] != "12345" {
		t.Errorf("id = %v, want 12345", m["id"])
	}
	if m["title"] != "Test Page" {
		t.Errorf("title = %v, want Test Page", m["title"])
	}
}

func TestSchema_GetBySpaceTitle(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// First call: resolve space key
		if strings.Contains(r.URL.Path, "spaces") {
			json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Space]{
				Results: []confluence.Space{{ID: "100", Key: "DEV"}},
			})
			return
		}
		// Second call: list pages with title filter
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Page]{
			Results: []confluence.Page{{ID: "42", Title: "My Page", Status: "current"}},
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `get(space=DEV, title="My Page"){minimal}`)

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["id"] != "42" {
		t.Errorf("id = %v, want 42", m["id"])
	}
}

func TestSchema_Spaces(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Space]{
			Results: []confluence.Space{
				{ID: "1", Key: "DEV", Name: "Development"},
				{ID: "2", Key: "OPS", Name: "Operations"},
			},
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `spaces(){minimal}`)

	var arr []map[string]any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(arr))
	}
	if arr[0]["key"] != "DEV" {
		t.Errorf("space[0].key = %v, want DEV", arr[0]["key"])
	}
}

func TestSchema_Children(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Page]{
			Results: []confluence.Page{
				{ID: "101", Title: "Child 1"},
			},
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `children(100){minimal}`)

	var arr []map[string]any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 child, got %d", len(arr))
	}
	if arr[0]["id"] != "101" {
		t.Errorf("child.id = %v, want 101", arr[0]["id"])
	}
}

func TestSchema_Search(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		if cql != "type=page" {
			t.Errorf("unexpected CQL: %s", cql)
		}
		json.NewEncoder(w).Encode(confluence.SearchResult{
			Results: []confluence.SearchResultItem{
				{
					Title:   "Found",
					Content: &confluence.V1Content{ID: "77", Type: "page"},
				},
			},
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `search("type=page"){default}`)

	var arr []map[string]any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 1 || arr[0]["id"] != "77" {
		t.Errorf("unexpected search results: %+v", arr)
	}
}

func TestSchema_Ancestors(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Ancestor]{
			Results: []confluence.Ancestor{
				{ID: "1", Title: "Root"},
			},
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `ancestors(100){minimal}`)

	var arr []map[string]any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 1 || arr[0]["Title"] != "Root" {
		// Ancestors returns raw confluence.Ancestor struct, fields are PascalCase from JSON tags
		// Actually, let's check — Ancestor has json tags "id" and "title"
		if arr[0]["title"] != "Root" {
			t.Errorf("unexpected ancestors: %+v", arr)
		}
	}
}

func TestSchema_Batch(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/spaces" {
			json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Space]{
				Results: []confluence.Space{{ID: "1", Key: "DEV"}},
			})
			return
		}
		json.NewEncoder(w).Encode(confluence.Page{ID: "12345", Title: "P"})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `spaces(){minimal}; get(12345){minimal}`)

	var arr []any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 results, got %d", len(arr))
	}
}

func TestSchema_HistoryNotImplemented(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()

	schema := NewSchema(client)
	// history returns an error, but in agentquery batch mode it's inlined.
	// For a single statement, QueryJSONWithMode wraps it as an error.
	data, err := schema.QueryJSONWithMode(`history(12345){minimal}`, agentquery.HumanReadable)
	if err != nil {
		// Single-statement error is fine — the error is returned directly
		if !strings.Contains(err.Error(), "not yet implemented") {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}
	// If we got data instead, check that it contains an error
	if !strings.Contains(string(data), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' in result, got: %s", data)
	}
}

func TestSchema_FieldPresets(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.Page{
			ID:     "1",
			Title:  "Test",
			Status: "current",
		})
	})
	defer ts.Close()

	schema := NewSchema(client)

	tests := []struct {
		preset string
		count  int
	}{
		{"minimal", 3},     // id, title, status
		{"default", 6},     // id, title, status, spaceKey, version, url
		{"overview", 8},    // id, title, status, spaceKey, version, ancestors, labels, url
		{"full", 12},       // id, title, status, spaceKey, version, ancestors, labels, body, created, updated, author, url
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			result := queryJSON(t, schema, `get(1){`+tt.preset+`}`)
			var m map[string]any
			if err := json.Unmarshal([]byte(result), &m); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(m) != tt.count {
				t.Errorf("preset %q: expected %d fields, got %d: %v", tt.preset, tt.count, len(m), m)
			}
		})
	}
}

func TestSchema_InvalidOperation(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()

	schema := NewSchema(client)
	_, err := schema.QueryJSONWithMode(`bogus(12345)`, agentquery.HumanReadable)
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
}

func TestSchema_InvalidField(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()

	schema := NewSchema(client)
	_, err := schema.QueryJSONWithMode(`get(1){bogusfield}`, agentquery.HumanReadable)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestSchema_AllOperationsRegistered(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Return something valid for any request
		json.NewEncoder(w).Encode(confluence.Page{ID: "1", Title: "T"})
	})
	defer ts.Close()

	schema := NewSchema(client)
	ops := []string{"get", "list", "search", "children", "ancestors", "tree", "spaces", "history", "schema"}
	for _, op := range ops {
		// Just verify the operation is recognized by the parser (no parse error).
		// Some will fail at execution (missing args), that's fine.
		_, err := schema.QueryJSONWithMode(op+`(12345){minimal}`, agentquery.HumanReadable)
		// For schema(), the positional arg 12345 is just ignored.
		// For other ops, they may return API errors but should not return "unknown operation".
		if err != nil && strings.Contains(err.Error(), "unknown operation") {
			t.Errorf("operation %q should be registered, got: %v", op, err)
		}
	}
}

func TestSchema_SchemaIntrospection(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `schema()`)

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Should have operations, fields, presets, defaultFields
	if _, ok := m["operations"]; !ok {
		t.Error("schema() missing 'operations'")
	}
	if _, ok := m["fields"]; !ok {
		t.Error("schema() missing 'fields'")
	}
	if _, ok := m["presets"]; !ok {
		t.Error("schema() missing 'presets'")
	}
}

func TestSchema_CompactFormat(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.Page{
			ID:    "1",
			Title: "Test",
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	data, err := schema.QueryJSONWithMode(`get(1){minimal}`, agentquery.LLMReadable)
	if err != nil {
		t.Fatalf("compact query failed: %v", err)
	}

	// Compact format should not be valid JSON (it's tabular).
	result := string(data)
	if strings.HasPrefix(strings.TrimSpace(result), "{") || strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Errorf("compact output looks like JSON: %s", result)
	}
}

func TestSchema_DefaultFieldsUsed(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.Page{
			ID:      "1",
			Title:   "T",
			Status:  "current",
			Version: &confluence.Version{Number: 1},
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `get(1)`)

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// No projection specified — should use "default" preset (6 fields: id, title, status, spaceKey, version, url).
	if len(m) != 6 {
		t.Errorf("expected 6 fields from default preset, got %d: %v", len(m), m)
	}
}

func TestSchema_ApplyPageFields_NilSafety(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Page with all nil optional fields.
		json.NewEncoder(w).Encode(confluence.Page{
			ID:    "1",
			Title: "Bare",
		})
	})
	defer ts.Close()

	schema := NewSchema(client)
	result := queryJSON(t, schema, `get(1){full}`)

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// version, body, labels should be nil (not crash).
	if m["id"] != "1" {
		t.Errorf("id = %v, want 1", m["id"])
	}
}
