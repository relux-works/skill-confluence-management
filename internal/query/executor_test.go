package query

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ivalx1s/skill-confluence-manager/internal/confluence"
)

func newExecTestServer(handler http.HandlerFunc) (*httptest.Server, *Executor) {
	ts := httptest.NewServer(handler)
	client, _ := confluence.NewClient(confluence.Config{
		BaseURL:      ts.URL,
		Email:        "test@test.com",
		Token:        "tok",
		InstanceType: confluence.InstanceCloud,
		AuthType:     confluence.AuthBasic,
	})
	client.SetHTTPClient(ts.Client())
	return ts, NewExecutor(client)
}

func TestExecutor_GetByID(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.Page{
			ID:    "12345",
			Title: "Test Page",
		})
	})
	defer ts.Close()

	q, err := ParseQuery(`get(12345){minimal}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	m, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %T", results[0])
	}
	if m["id"] != "12345" {
		t.Errorf("id = %v, want 12345", m["id"])
	}
	if m["title"] != "Test Page" {
		t.Errorf("title = %v, want Test Page", m["title"])
	}
}

func TestExecutor_Spaces(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Space]{
			Results: []confluence.Space{
				{ID: "1", Key: "DEV", Name: "Development"},
				{ID: "2", Key: "OPS", Name: "Operations"},
			},
		})
	})
	defer ts.Close()

	q, err := ParseQuery(`spaces(){minimal}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	arr, ok := results[0].([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", results[0])
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(arr))
	}
}

func TestExecutor_Children(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Page]{
			Results: []confluence.Page{
				{ID: "101", Title: "Child 1"},
			},
		})
	})
	defer ts.Close()

	q, err := ParseQuery(`children(100){minimal}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	arr, ok := results[0].([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", results[0])
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 child, got %d", len(arr))
	}
}

func TestExecutor_Search(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {
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

	q, err := ParseQuery(`search("type=page"){default}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	arr, ok := results[0].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected []map, got %T", results[0])
	}
	if len(arr) != 1 || arr[0]["id"] != "77" {
		t.Errorf("unexpected search results: %+v", arr)
	}
}

func TestExecutor_Ancestors(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Ancestor]{
			Results: []confluence.Ancestor{
				{ID: "1", Title: "Root"},
			},
		})
	})
	defer ts.Close()

	q, err := ParseQuery(`ancestors(100){minimal}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	ancs, ok := results[0].([]confluence.Ancestor)
	if !ok {
		t.Fatalf("expected []Ancestor, got %T", results[0])
	}
	if len(ancs) != 1 || ancs[0].Title != "Root" {
		t.Errorf("unexpected: %+v", ancs)
	}
}

func TestExecutor_Batch(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Serve both spaces and get
		if r.URL.Path == "/api/v2/spaces" {
			json.NewEncoder(w).Encode(confluence.CursorPage[confluence.Space]{
				Results: []confluence.Space{{ID: "1", Key: "DEV"}},
			})
			return
		}
		json.NewEncoder(w).Encode(confluence.Page{ID: "12345", Title: "P"})
	})
	defer ts.Close()

	q, err := ParseQuery(`spaces(){minimal}; get(12345){minimal}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestExecutor_HistoryNotImplemented(t *testing.T) {
	ts, exec := newExecTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()

	q, err := ParseQuery(`history(12345){minimal}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, err = exec.Execute(q)
	if err == nil {
		t.Fatal("expected error for unimplemented history")
	}
}

func TestApplyPageFields(t *testing.T) {
	page := &confluence.Page{
		ID:     "1",
		Title:  "Test",
		Status: "current",
		Body: &confluence.PageBody{
			Storage: &confluence.BodyRepresentation{Value: "<p>hello</p>"},
		},
		Version: &confluence.Version{Number: 5},
		Labels: &confluence.LabelArray{
			Results: []confluence.Label{{Name: "a"}, {Name: "b"}},
		},
	}

	m := applyPageFields(page, []string{"id", "title", "body", "version", "labels"})
	result := m.(map[string]interface{})

	if result["id"] != "1" {
		t.Errorf("id = %v", result["id"])
	}
	if result["body"] != "<p>hello</p>" {
		t.Errorf("body = %v", result["body"])
	}
	if result["version"] != 5 {
		t.Errorf("version = %v", result["version"])
	}
	labels, ok := result["labels"].([]string)
	if !ok || len(labels) != 2 {
		t.Errorf("labels = %v", result["labels"])
	}
}

func TestApplyPageFields_DefaultPreset(t *testing.T) {
	page := &confluence.Page{
		ID:      "1",
		Title:   "T",
		Status:  "current",
		Version: &confluence.Version{Number: 1},
	}
	m := applyPageFields(page, nil)
	result := m.(map[string]interface{})
	// nil fields should use default preset (6 fields: id, title, status, spaceKey, version, url)
	if len(result) != 6 {
		t.Errorf("expected 6 fields from default preset, got %d: %v", len(result), result)
	}
}

func TestHelpers(t *testing.T) {
	args := []Arg{
		{Value: "positional0"},
		{Key: "named1", Value: "val1"},
		{Value: "positional1"},
	}

	if v := getPositionalArg(args, 0); v != "positional0" {
		t.Errorf("positional[0] = %q", v)
	}
	if v := getPositionalArg(args, 1); v != "positional1" {
		t.Errorf("positional[1] = %q", v)
	}
	if v := getPositionalArg(args, 2); v != "" {
		t.Errorf("positional[2] should be empty, got %q", v)
	}
	if v := getNamedArg(args, "named1"); v != "val1" {
		t.Errorf("named1 = %q", v)
	}
	if v := getNamedArg(args, "missing"); v != "" {
		t.Errorf("missing should be empty, got %q", v)
	}

	if !containsField([]string{"a", "b"}, "a") {
		t.Error("containsField should find 'a'")
	}
	if containsField([]string{"a", "b"}, "c") {
		t.Error("containsField should not find 'c'")
	}
	if containsField(nil, "a") {
		t.Error("containsField(nil) should return false")
	}
}
