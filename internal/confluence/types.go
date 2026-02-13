package confluence

import "encoding/json"

// --- Client Config ---

// InstanceType represents the Confluence deployment type.
type InstanceType string

const (
	InstanceCloud  InstanceType = "cloud"  // Confluence Cloud (*.atlassian.net)
	InstanceServer InstanceType = "server" // Confluence Server / Data Center
)

// AuthType represents the authentication method.
type AuthType string

const (
	AuthBasic  AuthType = "basic"  // Cloud: email + API token -> Basic base64(email:token)
	AuthBearer AuthType = "bearer" // Server/DC: Personal Access Token -> Bearer <token>
)

// Config holds the configuration needed to connect to a Confluence instance.
type Config struct {
	BaseURL      string       // e.g. "https://mycompany.atlassian.net/wiki" (Cloud) or "https://confluence.company.com" (Server)
	Email        string       // Required for Basic auth (Cloud)
	Token        string       // API token (Basic) or PAT (Bearer)
	InstanceType InstanceType // "cloud" or "server"
	AuthType     AuthType     // "basic" or "bearer"
}

// --- Error Types ---

// APIError represents an error response from the Confluence REST API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message,omitempty"`
	// V1 error format
	ErrorMessage string `json:"errorMessage,omitempty"`
	StatusText   string `json:"statusCode,omitempty"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.ErrorMessage != "" {
		return e.ErrorMessage
	}
	return "confluence: unknown API error"
}

// --- Page ---

// Page represents a Confluence page (v2 API response shape).
type Page struct {
	ID         string      `json:"id"`
	Status     string      `json:"status,omitempty"`     // "current", "draft", "trashed"
	Title      string      `json:"title,omitempty"`
	SpaceID    string      `json:"spaceId,omitempty"`
	ParentID   string      `json:"parentId,omitempty"`
	ParentType string      `json:"parentType,omitempty"`
	AuthorID   string      `json:"authorId,omitempty"`
	CreatedAt  string      `json:"createdAt,omitempty"`
	Version    *Version    `json:"version,omitempty"`
	Body       *PageBody   `json:"body,omitempty"`
	Labels     *LabelArray `json:"labels,omitempty"`
	Links      *PageLinks  `json:"_links,omitempty"`
}

// WebURL returns the web URL for this page, if available.
func (p *Page) WebURL() string {
	if p.Links != nil {
		return p.Links.WebUI
	}
	return ""
}

// PageBody holds page content in various formats.
type PageBody struct {
	Storage        *BodyRepresentation `json:"storage,omitempty"`
	AtlasDocFormat *BodyRepresentation `json:"atlas_doc_format,omitempty"`
}

// BodyRepresentation holds the actual content value.
type BodyRepresentation struct {
	Value          string `json:"value,omitempty"`
	Representation string `json:"representation,omitempty"`
}

// PageLinks holds link data from the API response.
type PageLinks struct {
	WebUI  string `json:"webui,omitempty"`
	EditUI string `json:"editui,omitempty"`
	TinyUI string `json:"tinyui,omitempty"`
}

// --- Version ---

// Version represents a page version.
type Version struct {
	Number    int    `json:"number,omitempty"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	AuthorID  string `json:"authorId,omitempty"`
}

// --- Space ---

// Space represents a Confluence space.
type Space struct {
	ID          string      `json:"id"`
	Key         string      `json:"key,omitempty"`
	Name        string      `json:"name,omitempty"`
	Type        string      `json:"type,omitempty"`   // "global", "personal"
	Status      string      `json:"status,omitempty"` // "current", "archived"
	Description *SpaceDesc  `json:"description,omitempty"`
	HomepageID  string      `json:"homepageId,omitempty"`
	Links       *SpaceLinks `json:"_links,omitempty"`
}

// SpaceDesc holds space description.
type SpaceDesc struct {
	Plain *BodyRepresentation `json:"plain,omitempty"`
}

// SpaceLinks holds link data for a space.
type SpaceLinks struct {
	WebUI string `json:"webui,omitempty"`
}

// --- Label ---

// Label represents a Confluence label.
type Label struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Prefix string `json:"prefix,omitempty"` // "global", "my", "team"
}

// LabelArray is a wrapper for label results.
type LabelArray struct {
	Results []Label `json:"results,omitempty"`
}

// --- Ancestor (breadcrumb) ---

// Ancestor represents a page ancestor in the hierarchy.
type Ancestor struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// --- Search ---

// SearchResult represents a CQL search result (v1 API).
type SearchResult struct {
	Results        []SearchResultItem `json:"results"`
	Start          int                `json:"start"`
	Limit          int                `json:"limit"`
	Size           int                `json:"size"`
	TotalSize      int                `json:"totalSize,omitempty"`
	CQLQuery       string             `json:"cqlQuery,omitempty"`
	Links          json.RawMessage    `json:"_links,omitempty"`
}

// SearchResultItem is a single item in CQL search results.
type SearchResultItem struct {
	Content   *V1Content      `json:"content,omitempty"`
	Title     string          `json:"title,omitempty"`
	Excerpt   string          `json:"excerpt,omitempty"`
	URL       string          `json:"url,omitempty"`
	ResultGlobalContainer *ResultContainer `json:"resultGlobalContainer,omitempty"`
}

// ResultContainer holds space info for a search result.
type ResultContainer struct {
	Title      string `json:"title,omitempty"`
	DisplayURL string `json:"displayUrl,omitempty"`
}

// --- V1 Content (for search results and Server/DC) ---

// V1Content represents a Confluence content object (v1 API).
type V1Content struct {
	ID         string          `json:"id"`
	Type       string          `json:"type,omitempty"`   // "page", "blogpost", "comment"
	Status     string          `json:"status,omitempty"` // "current", "draft"
	Title      string          `json:"title,omitempty"`
	Space      *V1Space        `json:"space,omitempty"`
	Body       *V1Body         `json:"body,omitempty"`
	Version    *V1Version      `json:"version,omitempty"`
	Ancestors  []V1Content     `json:"ancestors,omitempty"`
	Children   *V1Children     `json:"children,omitempty"`
	Metadata   *V1Metadata     `json:"metadata,omitempty"`
	Links      json.RawMessage `json:"_links,omitempty"`
	Expandable json.RawMessage `json:"_expandable,omitempty"`
}

// V1Space represents a space in v1 API.
type V1Space struct {
	ID   int    `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// V1Body holds body content in v1 API.
type V1Body struct {
	Storage *V1BodyContent `json:"storage,omitempty"`
	View    *V1BodyContent `json:"view,omitempty"`
}

// V1BodyContent holds the value and representation.
type V1BodyContent struct {
	Value          string `json:"value,omitempty"`
	Representation string `json:"representation,omitempty"`
}

// V1Version represents version info in v1 API.
type V1Version struct {
	Number  int    `json:"number,omitempty"`
	Message string `json:"message,omitempty"`
	When    string `json:"when,omitempty"`
	By      *V1User `json:"by,omitempty"`
}

// V1User represents a user in v1 API.
type V1User struct {
	AccountID   string `json:"accountId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Username    string `json:"username,omitempty"` // Server/DC
}

// V1Children holds child content references.
type V1Children struct {
	Page *V1PageResults `json:"page,omitempty"`
}

// V1PageResults is a paginated list of v1 content.
type V1PageResults struct {
	Results []V1Content `json:"results,omitempty"`
	Start   int         `json:"start,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Size    int         `json:"size,omitempty"`
}

// V1Metadata holds metadata (labels etc.) in v1 API.
type V1Metadata struct {
	Labels *V1LabelResults `json:"labels,omitempty"`
}

// V1LabelResults is a paginated list of labels.
type V1LabelResults struct {
	Results []Label `json:"results,omitempty"`
	Start   int     `json:"start,omitempty"`
	Limit   int     `json:"limit,omitempty"`
	Size    int     `json:"size,omitempty"`
}

// --- Pagination (v2 cursor-based) ---

// CursorPage is the generic wrapper for v2 cursor-paginated responses.
type CursorPage[T any] struct {
	Results []T            `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
}

// PaginationLinks holds pagination cursors.
type PaginationLinks struct {
	Next string `json:"next,omitempty"`
}

// HasMore returns true if there are more pages.
func (p *CursorPage[T]) HasMore() bool {
	return p.Links != nil && p.Links.Next != ""
}

// --- Request types for write operations ---

// CreatePageRequest is the v2 request body for creating a page.
type CreatePageRequest struct {
	SpaceID  string              `json:"spaceId"`
	Status   string              `json:"status,omitempty"` // "current" or "draft"
	Title    string              `json:"title"`
	ParentID string              `json:"parentId,omitempty"`
	Body     *CreatePageBody     `json:"body,omitempty"`
}

// CreatePageBody holds the body for page creation.
type CreatePageBody struct {
	Representation string `json:"representation"` // "storage" or "atlas_doc_format"
	Value          string `json:"value"`
}

// UpdatePageRequest is the v2 request body for updating a page.
type UpdatePageRequest struct {
	ID      string          `json:"id"`
	Status  string          `json:"status"` // "current"
	Title   string          `json:"title"`
	Body    *CreatePageBody `json:"body,omitempty"`
	Version *VersionUpdate  `json:"version"`
}

// VersionUpdate holds the version number for page updates.
type VersionUpdate struct {
	Number  int    `json:"number"`
	Message string `json:"message,omitempty"`
}

// --- Label operations ---

// AddLabelsRequest is the v2 request body for adding labels.
type AddLabelsRequest []AddLabelEntry

// AddLabelEntry is a single label to add.
type AddLabelEntry struct {
	Prefix string `json:"prefix"` // "global"
	Name   string `json:"name"`
}
