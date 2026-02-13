package query

import (
	"fmt"

	"github.com/ivalx1s/skill-confluence-manager/internal/confluence"
)

// Executor runs parsed DSL queries against the Confluence API.
type Executor struct {
	client *confluence.Client
}

// NewExecutor creates an executor with the given Confluence client.
func NewExecutor(client *confluence.Client) *Executor {
	return &Executor{client: client}
}

// Execute runs all statements in a query and returns results.
func (e *Executor) Execute(q *Query) ([]interface{}, error) {
	var results []interface{}
	for _, stmt := range q.Statements {
		result, err := e.execStatement(stmt)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", stmt.Operation, err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (e *Executor) execStatement(stmt Statement) (interface{}, error) {
	switch stmt.Operation {
	case "get":
		return e.execGet(stmt)
	case "list":
		return e.execList(stmt)
	case "search":
		return e.execSearch(stmt)
	case "children":
		return e.execChildren(stmt)
	case "ancestors":
		return e.execAncestors(stmt)
	case "tree":
		return e.execTree(stmt)
	case "spaces":
		return e.execSpaces(stmt)
	case "history":
		return e.execHistory(stmt)
	default:
		return nil, fmt.Errorf("unknown operation %q", stmt.Operation)
	}
}

// --- Operation handlers ---

func (e *Executor) execGet(stmt Statement) (interface{}, error) {
	// get(PAGE_ID) or get(space=KEY, title="Page Title")
	pageID := getPositionalArg(stmt.Args, 0)
	if pageID != "" {
		includeBody := containsField(stmt.Fields, "body")
		page, err := e.client.GetPage(pageID, includeBody)
		if err != nil {
			return nil, err
		}
		return applyPageFields(page, stmt.Fields), nil
	}

	// get by space+title
	spaceKey := getNamedArg(stmt.Args, "space")
	title := getNamedArg(stmt.Args, "title")
	if spaceKey != "" && title != "" {
		pages, err := e.client.ListPages(spaceKey, title, 1)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("page %q not found in space %s", title, spaceKey)
		}
		return applyPageFields(&pages[0], stmt.Fields), nil
	}

	return nil, fmt.Errorf("get requires a page ID or space+title args")
}

func (e *Executor) execList(stmt Statement) (interface{}, error) {
	spaceKey := getNamedArg(stmt.Args, "space")
	if spaceKey == "" {
		return nil, fmt.Errorf("list requires space=KEY")
	}

	label := getNamedArg(stmt.Args, "label")
	if label != "" {
		// Labels require CQL search (v2 doesn't support label filter).
		cql := fmt.Sprintf("type=page AND space=%q AND label=%q", spaceKey, label)
		result, err := e.client.SearchCQL(cql, 50)
		if err != nil {
			return nil, err
		}
		return searchToMinimal(result), nil
	}

	title := getNamedArg(stmt.Args, "title")
	pages, err := e.client.ListPages(spaceKey, title, 0)
	if err != nil {
		return nil, err
	}

	var results []interface{}
	for i := range pages {
		results = append(results, applyPageFields(&pages[i], stmt.Fields))
	}
	return results, nil
}

func (e *Executor) execSearch(stmt Statement) (interface{}, error) {
	cql := getPositionalArg(stmt.Args, 0)
	if cql == "" {
		return nil, fmt.Errorf("search requires a CQL query string")
	}

	result, err := e.client.SearchCQL(cql, 25)
	if err != nil {
		return nil, err
	}
	return searchToMinimal(result), nil
}

func (e *Executor) execChildren(stmt Statement) (interface{}, error) {
	pageID := getPositionalArg(stmt.Args, 0)
	if pageID == "" {
		return nil, fmt.Errorf("children requires a page ID")
	}

	children, err := e.client.GetChildren(pageID, 0)
	if err != nil {
		return nil, err
	}

	var results []interface{}
	for i := range children {
		results = append(results, applyPageFields(&children[i], stmt.Fields))
	}
	return results, nil
}

func (e *Executor) execAncestors(stmt Statement) (interface{}, error) {
	pageID := getPositionalArg(stmt.Args, 0)
	if pageID == "" {
		return nil, fmt.Errorf("ancestors requires a page ID")
	}

	ancestors, err := e.client.GetAncestors(pageID)
	if err != nil {
		return nil, err
	}
	return ancestors, nil
}

func (e *Executor) execTree(stmt Statement) (interface{}, error) {
	pageID := getPositionalArg(stmt.Args, 0)
	if pageID == "" {
		return nil, fmt.Errorf("tree requires a page ID")
	}

	depthStr := getNamedArg(stmt.Args, "depth")
	maxDepth := 3 // default
	if depthStr != "" {
		fmt.Sscanf(depthStr, "%d", &maxDepth)
	}
	if maxDepth > 10 {
		maxDepth = 10
	}

	return e.buildTree(pageID, stmt.Fields, maxDepth, 0)
}

type treeNode struct {
	Page     interface{} `json:"page"`
	Children []treeNode  `json:"children,omitempty"`
}

func (e *Executor) buildTree(pageID string, fields []string, maxDepth, currentDepth int) (*treeNode, error) {
	page, err := e.client.GetPage(pageID, containsField(fields, "body"))
	if err != nil {
		return nil, err
	}

	node := &treeNode{Page: applyPageFields(page, fields)}

	if currentDepth >= maxDepth {
		return node, nil
	}

	children, err := e.client.GetChildren(pageID, 0)
	if err != nil {
		return node, nil // non-fatal
	}

	for _, child := range children {
		childNode, err := e.buildTree(child.ID, fields, maxDepth, currentDepth+1)
		if err != nil {
			continue
		}
		node.Children = append(node.Children, *childNode)
	}

	return node, nil
}

func (e *Executor) execSpaces(stmt Statement) (interface{}, error) {
	spaces, err := e.client.ListSpaces(0)
	if err != nil {
		return nil, err
	}

	var results []interface{}
	for _, s := range spaces {
		m := map[string]interface{}{
			"id":  s.ID,
			"key": s.Key,
		}
		if containsField(stmt.Fields, "name") || stmt.Fields == nil {
			m["name"] = s.Name
		}
		if containsField(stmt.Fields, "type") {
			m["type"] = s.Type
		}
		if containsField(stmt.Fields, "status") {
			m["status"] = s.Status
		}
		if containsField(stmt.Fields, "homepageId") {
			m["homepageId"] = s.HomepageID
		}
		results = append(results, m)
	}
	return results, nil
}

func (e *Executor) execHistory(stmt Statement) (interface{}, error) {
	// History is not yet implemented in the client — return a placeholder.
	return nil, fmt.Errorf("history operation not yet implemented")
}

// --- Helpers ---

func getPositionalArg(args []Arg, idx int) string {
	count := 0
	for _, a := range args {
		if a.Key == "" {
			if count == idx {
				return a.Value
			}
			count++
		}
	}
	return ""
}

func getNamedArg(args []Arg, name string) string {
	for _, a := range args {
		if a.Key == name {
			return a.Value
		}
	}
	return ""
}

func containsField(fields []string, name string) bool {
	if fields == nil {
		return false
	}
	for _, f := range fields {
		if f == name {
			return true
		}
	}
	return false
}

func applyPageFields(page *confluence.Page, fields []string) interface{} {
	if fields == nil {
		// Default fields
		fields = FieldPresets["default"]
	}

	m := make(map[string]interface{})
	for _, f := range fields {
		switch f {
		case "id":
			m["id"] = page.ID
		case "title":
			m["title"] = page.Title
		case "status":
			m["status"] = page.Status
		case "spaceId":
			m["spaceId"] = page.SpaceID
		case "spaceKey":
			m["spaceKey"] = page.SpaceID // best effort — v2 returns spaceId
		case "version":
			if page.Version != nil {
				m["version"] = page.Version.Number
			}
		case "body":
			if page.Body != nil && page.Body.Storage != nil {
				m["body"] = page.Body.Storage.Value
			}
		case "labels":
			if page.Labels != nil {
				names := make([]string, len(page.Labels.Results))
				for i, l := range page.Labels.Results {
					names[i] = l.Name
				}
				m["labels"] = names
			}
		case "created":
			m["created"] = page.CreatedAt
		case "author":
			m["author"] = page.AuthorID
		case "url":
			m["url"] = page.WebURL()
		case "parentId":
			m["parentId"] = page.ParentID
		case "ancestors":
			// Not included in basic page fetch — would need separate call
			m["ancestors"] = nil
		}
	}
	return m
}

func searchToMinimal(result *confluence.SearchResult) interface{} {
	var items []map[string]interface{}
	for _, r := range result.Results {
		item := map[string]interface{}{
			"title":   r.Title,
			"excerpt": r.Excerpt,
		}
		if r.Content != nil {
			item["id"] = r.Content.ID
			item["type"] = r.Content.Type
			if r.Content.Space != nil {
				item["spaceKey"] = r.Content.Space.Key
			}
		}
		items = append(items, item)
	}
	return items
}
