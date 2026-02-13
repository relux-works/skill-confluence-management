# DSL Query Examples

## Grammar

```
batch     = query (";" query)*
query     = operation "(" args ")" [ "{" fields "}" ]
args      = arg ("," arg)*  |  ε
arg       = ident "=" value  |  value
fields    = ident+  |  preset_name
value     = ident | "quoted string"
```

## Operations

### get — single page

```bash
# By ID
confluence-mgmt q 'get(12345){full}'

# By ID, minimal fields
confluence-mgmt q 'get(12345){minimal}'

# By space + title
confluence-mgmt q 'get(space=DEV, title="Architecture Decision Records"){default}'
```

### list — pages in space

```bash
# All pages in space
confluence-mgmt q 'list(space=DEV){minimal}'

# With title filter
confluence-mgmt q 'list(space=DEV, title="API"){default}'

# By label (routes through CQL)
confluence-mgmt q 'list(space=DEV, label=api-docs){default}'
```

### search — CQL

```bash
# Full-text
confluence-mgmt q 'search("type=page AND text~\"migration\""){default}'

# Recently modified
confluence-mgmt q 'search("type=page AND space=DEV AND lastmodified >= now(\"-7d\")"){minimal}'

# By creator
confluence-mgmt q 'search("type=page AND creator=currentUser()"){default}'
```

### children / ancestors / tree

```bash
# Direct children
confluence-mgmt q 'children(12345){minimal}'

# Breadcrumbs
confluence-mgmt q 'ancestors(12345){minimal}'

# Recursive tree (default depth=3)
confluence-mgmt q 'tree(12345){minimal}'

# Custom depth
confluence-mgmt q 'tree(12345, depth=5){minimal}'
```

### spaces

```bash
confluence-mgmt q 'spaces(){minimal}'
confluence-mgmt q 'spaces(){default}'
```

### Batch

```bash
confluence-mgmt q 'spaces(){minimal}; list(space=DEV){default}; get(12345){overview}'
```

## Field Presets

| Preset | Includes |
|--------|----------|
| minimal | id, title, status |
| default | id, title, status, spaceKey, version, url |
| overview | + ancestors, labels |
| full | + body, created, updated, author |
