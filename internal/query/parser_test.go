package query

import (
	"testing"
)

func TestParseQuery_SimpleGet(t *testing.T) {
	q, err := ParseQuery(`get(12345){minimal}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(q.Statements))
	}
	s := q.Statements[0]
	if s.Operation != "get" {
		t.Errorf("operation = %q, want %q", s.Operation, "get")
	}
	if len(s.Args) != 1 || s.Args[0].Value != "12345" {
		t.Errorf("args = %+v, want positional 12345", s.Args)
	}
	if len(s.Fields) != 3 { // minimal = id, title, status
		t.Errorf("expected 3 fields from minimal preset, got %d: %v", len(s.Fields), s.Fields)
	}
}

func TestParseQuery_NamedArgs(t *testing.T) {
	q, err := ParseQuery(`get(space=DEV, title="My Page"){default}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := q.Statements[0]
	if len(s.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(s.Args))
	}
	if s.Args[0].Key != "space" || s.Args[0].Value != "DEV" {
		t.Errorf("arg[0] = %+v, want space=DEV", s.Args[0])
	}
	if s.Args[1].Key != "title" || s.Args[1].Value != "My Page" {
		t.Errorf("arg[1] = %+v, want title=My Page", s.Args[1])
	}
}

func TestParseQuery_Batch(t *testing.T) {
	q, err := ParseQuery(`spaces(){minimal}; get(12345){default}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(q.Statements))
	}
	if q.Statements[0].Operation != "spaces" {
		t.Errorf("stmt[0].operation = %q, want spaces", q.Statements[0].Operation)
	}
	if q.Statements[1].Operation != "get" {
		t.Errorf("stmt[1].operation = %q, want get", q.Statements[1].Operation)
	}
}

func TestParseQuery_AllOperations(t *testing.T) {
	ops := []string{"get", "list", "search", "children", "ancestors", "tree", "spaces", "history"}
	for _, op := range ops {
		_, err := ParseQuery(op + `(12345){minimal}`)
		if err != nil {
			t.Errorf("operation %q should be valid, got error: %v", op, err)
		}
	}
}

func TestParseQuery_InvalidOperation(t *testing.T) {
	_, err := ParseQuery(`bogus(12345)`)
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
}

func TestParseQuery_FieldPresets(t *testing.T) {
	tests := []struct {
		preset string
		count  int
	}{
		{"minimal", 3},
		{"default", 6},
		{"overview", 8},
		{"full", 12},
	}
	for _, tt := range tests {
		q, err := ParseQuery(`get(1){` + tt.preset + `}`)
		if err != nil {
			t.Errorf("preset %q: unexpected error: %v", tt.preset, err)
			continue
		}
		if len(q.Statements[0].Fields) != tt.count {
			t.Errorf("preset %q: expected %d fields, got %d: %v", tt.preset, tt.count, len(q.Statements[0].Fields), q.Statements[0].Fields)
		}
	}
}

func TestParseQuery_EmptyArgs(t *testing.T) {
	q, err := ParseQuery(`spaces()`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Statements[0].Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(q.Statements[0].Args))
	}
}

func TestParseQuery_QuotedCQL(t *testing.T) {
	q, err := ParseQuery(`search("type=page AND text~\"api\""){default}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := q.Statements[0]
	if s.Args[0].Value != `type=page AND text~\"api\"` {
		t.Errorf("CQL = %q", s.Args[0].Value)
	}
}

func TestParseQuery_EmptyInput(t *testing.T) {
	_, err := ParseQuery("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseQuery_InvalidField(t *testing.T) {
	_, err := ParseQuery(`get(1){bogusfield}`)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestParseQuery_DeduplicateFields(t *testing.T) {
	q, err := ParseQuery(`get(1){id title id}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Statements[0].Fields) != 2 {
		t.Errorf("expected 2 unique fields, got %d", len(q.Statements[0].Fields))
	}
}
