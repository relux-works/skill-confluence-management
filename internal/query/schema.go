// Package query wires up the Confluence domain types against the agentquery library.
// It replaces the previous hand-rolled parser and executor with agentquery.Schema[*confluence.Page].
package query

import (
	"fmt"

	"github.com/ivalx1s/skill-agent-facing-api/agentquery"
	"github.com/ivalx1s/skill-confluence-manager/internal/confluence"
)

// treeNode is the recursive structure returned by the tree() operation.
type treeNode struct {
	Page     any        `json:"page"`
	Children []treeNode `json:"children,omitempty"`
}

// NewSchema builds a fully configured agentquery.Schema for Confluence pages.
// The client is captured by closure in all operation handlers.
func NewSchema(client *confluence.Client) *agentquery.Schema[*confluence.Page] {
	schema := agentquery.NewSchema[*confluence.Page]()

	// --- Page fields ---
	schema.Field("id", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.ID
	})
	schema.Field("title", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.Title
	})
	schema.Field("status", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.Status
	})
	schema.Field("spaceId", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.SpaceID
	})
	schema.Field("spaceKey", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		// v2 returns spaceId, not spaceKey — best effort
		return p.SpaceID
	})
	schema.Field("version", func(p *confluence.Page) any {
		if p == nil || p.Version == nil {
			return nil
		}
		return p.Version.Number
	})
	schema.Field("body", func(p *confluence.Page) any {
		if p == nil || p.Body == nil || p.Body.Storage == nil {
			return nil
		}
		return p.Body.Storage.Value
	})
	schema.Field("labels", func(p *confluence.Page) any {
		if p == nil || p.Labels == nil {
			return nil
		}
		names := make([]string, len(p.Labels.Results))
		for i, l := range p.Labels.Results {
			names[i] = l.Name
		}
		return names
	})
	schema.Field("created", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.CreatedAt
	})
	schema.Field("updated", func(p *confluence.Page) any {
		// Page struct doesn't have UpdatedAt — return nil
		return nil
	})
	schema.Field("author", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.AuthorID
	})
	schema.Field("url", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.WebURL()
	})
	schema.Field("parentId", func(p *confluence.Page) any {
		if p == nil {
			return nil
		}
		return p.ParentID
	})
	schema.Field("ancestors", func(p *confluence.Page) any {
		// Not included in basic page fetch — would need separate call
		return nil
	})

	// Space-compatible fields (accessor returns nil for Pages, used to make parser accept them).
	schema.Field("key", func(p *confluence.Page) any { return nil })
	schema.Field("name", func(p *confluence.Page) any { return nil })
	schema.Field("type", func(p *confluence.Page) any { return nil })
	schema.Field("homepageId", func(p *confluence.Page) any { return nil })

	// --- Presets ---
	schema.Preset("minimal", "id", "title", "status")
	schema.Preset("default", "id", "title", "status", "spaceKey", "version", "url")
	schema.Preset("overview", "id", "title", "status", "spaceKey", "version", "ancestors", "labels", "url")
	schema.Preset("full", "id", "title", "status", "spaceKey", "version", "ancestors", "labels", "body", "created", "updated", "author", "url")

	// Default fields when no projection specified.
	schema.DefaultFields("default")

	// Loader is a no-op — Confluence operations call the API directly,
	// they don't load a bulk dataset. Individual handlers call the client.
	schema.SetLoader(func() ([]*confluence.Page, error) {
		return nil, nil
	})

	// --- Operations ---

	// get(PAGE_ID) or get(space=KEY, title="Title")
	schema.OperationWithMetadata("get", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opGet(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "Get page by ID or by space+title",
		Parameters: []agentquery.ParameterDef{
			{Name: "id", Type: "string", Optional: true, Description: "Page ID (positional)"},
			{Name: "space", Type: "string", Optional: true, Description: "Space key (use with title)"},
			{Name: "title", Type: "string", Optional: true, Description: "Page title (use with space)"},
		},
		Examples: []string{
			"get(12345) { default }",
			"get(space=DEV, title=\"My Page\") { full }",
		},
	})

	// list(space=KEY)
	schema.OperationWithMetadata("list", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opList(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "List pages in a space, with optional label/title filter",
		Parameters: []agentquery.ParameterDef{
			{Name: "space", Type: "string", Optional: false, Description: "Space key"},
			{Name: "label", Type: "string", Optional: true, Description: "Filter by label (uses CQL)"},
			{Name: "title", Type: "string", Optional: true, Description: "Filter by title substring"},
		},
		Examples: []string{
			"list(space=DEV) { default }",
			"list(space=DEV, label=api) { minimal }",
			"list(space=DEV, title=\"API\") { overview }",
		},
	})

	// search("CQL")
	schema.OperationWithMetadata("search", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opSearch(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "CQL search",
		Parameters: []agentquery.ParameterDef{
			{Name: "cql", Type: "string", Optional: false, Description: "CQL query string (positional)"},
		},
		Examples: []string{
			`search("type=page AND space=DEV") { default }`,
			`search("type=page AND text~\"API\"") { default }`,
		},
	})

	// children(PAGE_ID)
	schema.OperationWithMetadata("children", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opChildren(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "Direct children of a page",
		Parameters: []agentquery.ParameterDef{
			{Name: "id", Type: "string", Optional: false, Description: "Page ID (positional)"},
		},
		Examples: []string{
			"children(12345) { minimal }",
		},
	})

	// ancestors(PAGE_ID)
	schema.OperationWithMetadata("ancestors", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opAncestors(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "Breadcrumb chain (ancestor pages)",
		Parameters: []agentquery.ParameterDef{
			{Name: "id", Type: "string", Optional: false, Description: "Page ID (positional)"},
		},
		Examples: []string{
			"ancestors(12345) { minimal }",
		},
	})

	// tree(PAGE_ID)
	schema.OperationWithMetadata("tree", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opTree(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "Recursive children tree with configurable depth",
		Parameters: []agentquery.ParameterDef{
			{Name: "id", Type: "string", Optional: false, Description: "Page ID (positional)"},
			{Name: "depth", Type: "int", Optional: true, Default: 3, Description: "Max recursion depth (max 10)"},
		},
		Examples: []string{
			"tree(12345) { minimal }",
			"tree(12345, depth=5) { default }",
		},
	})

	// spaces()
	schema.OperationWithMetadata("spaces", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return opSpaces(ctx, client)
	}, agentquery.OperationMetadata{
		Description: "List all accessible spaces",
		Examples: []string{
			"spaces() { minimal }",
			"spaces()",
		},
	})

	// history(PAGE_ID)
	schema.OperationWithMetadata("history", func(ctx agentquery.OperationContext[*confluence.Page]) (any, error) {
		return nil, fmt.Errorf("history operation not yet implemented")
	}, agentquery.OperationMetadata{
		Description: "Version history (not yet implemented)",
		Parameters: []agentquery.ParameterDef{
			{Name: "id", Type: "string", Optional: false, Description: "Page ID (positional)"},
		},
	})

	return schema
}

// --- Operation handlers ---

func opGet(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	pageID := getPositionalArg(ctx.Statement.Args, 0)
	if pageID != "" {
		includeBody := containsField(ctx.Statement.Fields, "body")
		page, err := client.GetPage(pageID, includeBody)
		if err != nil {
			return nil, err
		}
		return ctx.Selector.Apply(page), nil
	}

	// get by space+title
	spaceKey := getNamedArg(ctx.Statement.Args, "space")
	title := getNamedArg(ctx.Statement.Args, "title")
	if spaceKey != "" && title != "" {
		pages, err := client.ListPages(spaceKey, title, 1)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("page %q not found in space %s", title, spaceKey)
		}
		return ctx.Selector.Apply(&pages[0]), nil
	}

	return nil, fmt.Errorf("get requires a page ID or space+title args")
}

func opList(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	spaceKey := getNamedArg(ctx.Statement.Args, "space")
	if spaceKey == "" {
		return nil, fmt.Errorf("list requires space=KEY")
	}

	label := getNamedArg(ctx.Statement.Args, "label")
	if label != "" {
		// Labels require CQL search (v2 doesn't support label filter).
		cql := fmt.Sprintf("type=page AND space=%q AND label=%q", spaceKey, label)
		result, err := client.SearchCQL(cql, 50)
		if err != nil {
			return nil, err
		}
		return searchToMinimal(result), nil
	}

	title := getNamedArg(ctx.Statement.Args, "title")
	pages, err := client.ListPages(spaceKey, title, 0)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(pages))
	for i := range pages {
		results = append(results, ctx.Selector.Apply(&pages[i]))
	}
	return results, nil
}

func opSearch(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	cql := getPositionalArg(ctx.Statement.Args, 0)
	if cql == "" {
		return nil, fmt.Errorf("search requires a CQL query string")
	}

	result, err := client.SearchCQL(cql, 25)
	if err != nil {
		return nil, err
	}
	return searchToMinimal(result), nil
}

func opChildren(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	pageID := getPositionalArg(ctx.Statement.Args, 0)
	if pageID == "" {
		return nil, fmt.Errorf("children requires a page ID")
	}

	children, err := client.GetChildren(pageID, 0)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(children))
	for i := range children {
		results = append(results, ctx.Selector.Apply(&children[i]))
	}
	return results, nil
}

func opAncestors(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	pageID := getPositionalArg(ctx.Statement.Args, 0)
	if pageID == "" {
		return nil, fmt.Errorf("ancestors requires a page ID")
	}

	ancestors, err := client.GetAncestors(pageID)
	if err != nil {
		return nil, err
	}
	// Return raw Ancestor array (not Page), bypasses field selector.
	return ancestors, nil
}

func opTree(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	pageID := getPositionalArg(ctx.Statement.Args, 0)
	if pageID == "" {
		return nil, fmt.Errorf("tree requires a page ID")
	}

	depthStr := getNamedArg(ctx.Statement.Args, "depth")
	maxDepth := 3
	if depthStr != "" {
		fmt.Sscanf(depthStr, "%d", &maxDepth)
	}
	if maxDepth > 10 {
		maxDepth = 10
	}

	return buildTree(client, ctx.Selector, pageID, maxDepth, 0)
}

func buildTree(client *confluence.Client, selector *agentquery.FieldSelector[*confluence.Page], pageID string, maxDepth, currentDepth int) (*treeNode, error) {
	page, err := client.GetPage(pageID, false)
	if err != nil {
		return nil, err
	}

	node := &treeNode{Page: selector.Apply(page)}

	if currentDepth >= maxDepth {
		return node, nil
	}

	children, err := client.GetChildren(pageID, 0)
	if err != nil {
		return node, nil // non-fatal
	}

	for _, child := range children {
		childNode, err := buildTree(client, selector, child.ID, maxDepth, currentDepth+1)
		if err != nil {
			continue
		}
		node.Children = append(node.Children, *childNode)
	}

	return node, nil
}

func opSpaces(ctx agentquery.OperationContext[*confluence.Page], client *confluence.Client) (any, error) {
	spaces, err := client.ListSpaces(0)
	if err != nil {
		return nil, err
	}

	// Spaces are a different domain type — manually project fields.
	fields := ctx.Statement.Fields
	var results []map[string]any
	for _, s := range spaces {
		m := map[string]any{
			"id":  s.ID,
			"key": s.Key,
		}
		if containsField(fields, "name") || fields == nil {
			m["name"] = s.Name
		}
		if containsField(fields, "type") {
			m["type"] = s.Type
		}
		if containsField(fields, "status") {
			m["status"] = s.Status
		}
		if containsField(fields, "homepageId") {
			m["homepageId"] = s.HomepageID
		}
		results = append(results, m)
	}
	return results, nil
}

// --- Helpers ---

func getPositionalArg(args []agentquery.Arg, idx int) string {
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

func getNamedArg(args []agentquery.Arg, name string) string {
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

func searchToMinimal(result *confluence.SearchResult) any {
	var items []map[string]any
	for _, r := range result.Results {
		item := map[string]any{
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
