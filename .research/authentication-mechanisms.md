# Confluence API Authentication Mechanisms

Research date: 2026-02-13

---

## Overview

Atlassian provides multiple authentication methods for Confluence API access, varying by deployment model (Cloud vs. Server/Data Center). The most important finding for this project: **a single Atlassian API token works for both Jira and Confluence Cloud** -- no separate token is needed.

This document covers all available auth methods, compares them across deployment models, and provides setup guidance for building a CLI tool.

---

## Auth Methods Comparison

### Cloud

| Method | Auth Header | Token Source | Scope Control | Best For |
|--------|-------------|--------------|---------------|----------|
| **Basic Auth + API Token (classic)** | `Authorization: Basic base64(email:token)` | [id.atlassian.com](https://id.atlassian.com/manage-profile/security/api-tokens) | No -- inherits full user permissions | Scripts, CLI tools, personal automation |
| **Basic Auth + Scoped API Token** | `Authorization: Basic base64(email:token)` | [id.atlassian.com](https://id.atlassian.com/manage-profile/security/api-tokens) | Yes -- select specific scopes per product | Scripts needing least-privilege access |
| **OAuth 2.0 (3LO)** | `Authorization: Bearer <access_token>` | Developer console app | Yes -- granular scopes | Distributed apps, multi-user integrations |
| **Atlassian Connect (JWT)** | JWT in query string or header | Connect app descriptor | Connect scopes | Marketplace apps with UI modules |
| **Forge (OAuth 2.0 managed)** | Handled by Forge runtime | Forge app manifest | Forge permissions | Forge-native apps |

### Server / Data Center

| Method | Auth Header | Token Source | Best For |
|--------|-------------|--------------|----------|
| **Personal Access Token (PAT)** | `Authorization: Bearer <token>` | User profile > Settings > Personal access tokens | Scripts, integrations (recommended) |
| **Basic Auth (username:password)** | `Authorization: Basic base64(user:pass)` | User credentials | Legacy scripts (not recommended) |
| **OAuth 1.0a** | OAuth signature headers | Admin-configured application link | Third-party app integrations |
| **Cookie/Session Auth** | `Cookie: JSESSIONID=...` | Login endpoint | Browser-based or session-persistent tools |

> **Note:** Atlassian Server products reached end of support on February 15, 2024. Data Center remains supported.

---

## The Key Question: Jira Token Reuse

### YES -- Same Token Works for Both Jira and Confluence Cloud

This is confirmed by Atlassian's official documentation:

> "API tokens can be used with **Confluence Cloud, Jira Cloud, and Jira Align REST APIs**."
> -- [Manage API tokens for your Atlassian account](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/)

**How it works:**

1. API tokens are tied to your **Atlassian account** (managed at `id.atlassian.com`), not to individual products.
2. A single token authenticates you across all Cloud products your account has access to.
3. The token inherits whatever permissions your account has in each product.

**Two token types exist:**

| Token Type | Jira + Confluence? | Endpoint Pattern | Scope Control |
|------------|-------------------|------------------|---------------|
| **Classic (unscoped)** | Yes, same token | `https://<site>.atlassian.net/wiki/rest/api/...` (Confluence) or `https://<site>.atlassian.net/rest/api/...` (Jira) | No -- full user permissions |
| **Scoped** | Yes, same token, but scopes are per-product | `https://api.atlassian.com/ex/confluence/{cloudId}/...` or `https://api.atlassian.com/ex/jira/{cloudId}/...` | Yes -- select scopes per product at creation |

**Practical implication for CLI tool:** If a user already has an API token for Jira, they can reuse it for Confluence immediately. No new token generation needed.

### Server/Data Center: Separate Tokens

On Server/DC, PATs are **product-specific**. A Jira PAT does not work for Confluence and vice versa. Each product instance manages its own tokens independently.

---

## Setup Guide Per Method

### Method 1: Basic Auth with API Token (Cloud) -- RECOMMENDED FOR CLI

This is the simplest and most practical method for a CLI tool.

**Step 1: Generate API token**

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Give it a descriptive name (e.g., "confluence-cli")
4. Set expiration (1-365 days; defaults to 1 year after Dec 15, 2024)
5. Copy the token immediately -- it cannot be recovered later

**Step 2: Authenticate API requests**

```bash
# Using curl -u flag (simplest)
curl -u "your_email@example.com:YOUR_API_TOKEN" \
  -X GET \
  -H "Content-Type: application/json" \
  "https://your-domain.atlassian.net/wiki/rest/api/space"

# Using explicit Authorization header
ENCODED=$(echo -n "your_email@example.com:YOUR_API_TOKEN" | base64)
curl -X GET \
  -H "Authorization: Basic ${ENCODED}" \
  -H "Content-Type: application/json" \
  "https://your-domain.atlassian.net/wiki/rest/api/space"
```

**Required configuration values:**
- `email` -- Atlassian account email
- `api_token` -- Generated API token
- `domain` -- Atlassian site domain (e.g., `your-company.atlassian.net`)

**Endpoint format:** `https://{domain}/wiki/rest/api/{resource}`

> **Important:** Confluence Cloud allows anonymous access by default, so the server may not send an HTTP 401 challenge. Always include the Authorization header explicitly rather than relying on challenge-response.

---

### Method 2: Scoped API Token (Cloud)

For enhanced security with least-privilege access.

**Step 1: Generate scoped token**

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token **with scopes**"
3. Name it and set expiration
4. Select "Confluence" as the target application
5. Choose specific scopes (e.g., read content, write content)
6. Copy the token

**Step 2: Find your Cloud ID**

```bash
# Cloud ID is in the URL when you visit your site, or query it:
curl -u "email:token" \
  "https://your-domain.atlassian.net/_edge/tenant_info"
# Returns JSON with "cloudId" field
```

**Step 3: Make API requests**

```bash
# Scoped tokens MUST use api.atlassian.com endpoint
curl -u "your_email@example.com:YOUR_SCOPED_TOKEN" \
  -X GET \
  -H "Content-Type: application/json" \
  "https://api.atlassian.com/ex/confluence/{cloudId}/rest/api/space"
```

**Key difference from classic tokens:** Scoped tokens use `api.atlassian.com/ex/confluence/{cloudId}/` instead of `{domain}.atlassian.net/wiki/`.

---

### Method 3: OAuth 2.0 (3LO) (Cloud)

For distributed apps or multi-user scenarios.

**Step 1: Create an OAuth 2.0 app**

1. Go to https://developer.atlassian.com/console/myapps/
2. Create a new app
3. Add "OAuth 2.0 (3LO)" authorization
4. Configure callback URL
5. Add required Confluence scopes

**Step 2: Authorization flow**

```
GET https://auth.atlassian.com/authorize?
  audience=api.atlassian.com&
  client_id=YOUR_CLIENT_ID&
  scope=read:confluence-content.all write:confluence-content offline_access&
  redirect_uri=https://YOUR_CALLBACK_URL&
  state=RANDOM_STATE&
  response_type=code&
  prompt=consent
```

**Step 3: Exchange code for token**

```bash
curl --request POST \
  --url 'https://auth.atlassian.com/oauth/token' \
  --header 'Content-Type: application/json' \
  --data '{
    "grant_type": "authorization_code",
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "code": "AUTHORIZATION_CODE",
    "redirect_uri": "https://YOUR_CALLBACK_URL"
  }'
```

**Step 4: Get accessible resources (Cloud ID)**

```bash
curl --request GET \
  --url https://api.atlassian.com/oauth/token/accessible-resources \
  --header 'Authorization: Bearer ACCESS_TOKEN' \
  --header 'Accept: application/json'
```

**Step 5: Make API calls**

```bash
curl --request GET \
  --url "https://api.atlassian.com/ex/confluence/{cloudId}/rest/api/space" \
  --header 'Authorization: Bearer ACCESS_TOKEN' \
  --header 'Accept: application/json'
```

**Refresh tokens:** Include `offline_access` in scopes. Refresh tokens rotate on each use (90-day inactivity expiry, 10-minute reuse interval).

**Key Confluence OAuth scopes (classic):**

| Scope | Description |
|-------|-------------|
| `read:confluence-content.all` | Read all content including body |
| `read:confluence-content.summary` | Read content summaries only |
| `write:confluence-content` | Create/update pages, blogs, comments |
| `write:confluence-space` | Create/update/delete spaces |
| `write:confluence-file` | Upload attachments |
| `read:confluence-space.summary` | Read space info |
| `read:confluence-props` | Read content properties |
| `write:confluence-props` | Write content properties |
| `manage:confluence-configuration` | Manage global settings |
| `search:confluence` | Search content and spaces |
| `read:confluence-user` | View user info |
| `read:confluence-groups` | Read user groups |
| `write:confluence-groups` | Manage user groups |
| `readonly:content.attachment:confluence` | Download attachments |

---

### Method 4: Personal Access Token (Server/Data Center)

**Step 1: Create PAT**

1. In Confluence, click your avatar > Settings > Personal access tokens
2. Click "Create token"
3. Name the token and optionally set expiration
4. Copy the token

**Step 2: Use in API requests**

```bash
curl -H "Authorization: Bearer YOUR_PAT" \
  "https://confluence.your-company.com/rest/api/content"
```

**No email required** -- the PAT is self-contained (identifies the user by itself). This differs from Cloud API tokens which need the email:token pair.

**Admin management:** Admins can view, filter, and revoke tokens at Administration > Users & Security > Personal access tokens.

---

## Permission & Scope Models: Confluence vs. Jira

### Cloud -- API Token Permissions

- **Classic (unscoped) tokens** inherit the full permissions of the user account. If your account can edit a Confluence space, the token can too. Same for Jira.
- **Scoped tokens** let you restrict permissions at token creation. Scopes are product-specific:
  - Confluence scopes: `read:confluence-content.all`, `write:confluence-content`, etc.
  - Jira scopes: `read:jira-work`, `write:jira-work`, etc.
  - A single scoped token can include scopes for BOTH products.

### Cloud -- OAuth 2.0 Permissions

- Scopes are defined per app in the developer console.
- Confluence and Jira have separate scope namespaces (prefixed `confluence` vs `jira`).
- An app can request scopes for both products simultaneously.
- **Confluence permissions are not overridden by scopes** -- if a user lacks permission to a space, the scope won't bypass that.

### Server/Data Center

- PATs inherit the full permissions of the user who created them. No scope restrictions.
- Product-specific: a Confluence PAT only works on that Confluence instance.

---

## Recommendations for CLI Tool

### Primary Approach: Basic Auth + Classic API Token

**Why:**
1. Simplest setup (email + token + domain = done).
2. Same token works for both Jira and Confluence -- users with existing Jira setups don't need a new token.
3. No OAuth app registration, no callback URLs, no token refresh logic.
4. Sufficient for personal/team CLI use.

**Configuration the CLI should collect:**
```
confluence_domain: your-company.atlassian.net
email: user@example.com
api_token: <token>
```

**Support scoped tokens as well:** Same auth mechanism, just different base URL (`api.atlassian.com` + cloudId vs. `domain.atlassian.net`).

### Secondary Approach: PAT for Server/Data Center

**Why:** If the tool should support on-premises installations, PATs are the way.

**Configuration:**
```
confluence_url: https://confluence.your-company.com
pat: <token>
```

### Implementation Notes

1. **Token storage:** Store credentials securely. Consider OS keychain integration or at minimum an encrypted config file. Never store tokens in plaintext in a git-tracked location.
2. **Detect deployment type:** Cloud uses `*.atlassian.net` domains; Server/DC uses custom domains. The CLI can auto-detect based on the provided URL.
3. **Auth header construction:**
   - Cloud: `Authorization: Basic base64(email:token)`
   - Server/DC: `Authorization: Bearer <pat>`
4. **Rate limiting:** As of November 22, 2025, Atlassian enforces rate limits on API tokens. The CLI should handle 429 responses with exponential backoff.
5. **Token expiration:** Tokens expire (max 365 days for Cloud, configurable for DC). The CLI should handle 401 responses gracefully and prompt for token refresh.

---

## Sources

### Official Atlassian Documentation

- [Basic auth for REST APIs - Confluence Cloud](https://developer.atlassian.com/cloud/confluence/basic-auth-for-rest-apis/)
- [Using the REST API - Confluence Cloud](https://developer.atlassian.com/cloud/confluence/using-the-rest-api/)
- [Manage API tokens for your Atlassian account](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/)
- [Scoped API Tokens in Confluence Cloud](https://support.atlassian.com/confluence/kb/scoped-api-tokens-in-confluence-cloud/)
- [Understand user API tokens](https://support.atlassian.com/organization-administration/docs/understand-user-api-tokens/)
- [OAuth 2.0 (3LO) apps - Confluence Cloud](https://developer.atlassian.com/cloud/confluence/oauth-2-3lo-apps/)
- [Confluence scopes for OAuth 2.0 (3LO) and Forge apps](https://developer.atlassian.com/cloud/confluence/scopes-for-oauth-2-3LO-and-forge-apps/)
- [Jira scopes for OAuth 2.0 (3LO) and Forge apps](https://developer.atlassian.com/cloud/jira/platform/scopes-for-oauth-2-3LO-and-forge-apps/)
- [Using Personal Access Tokens - Enterprise Data Center](https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html)
- [Deprecation notice - Basic auth with passwords](https://developer.atlassian.com/cloud/confluence/deprecation-notice-basic-auth/)
- [Authentication and authorization for developers](https://developer.atlassian.com/developer-guide/auth/)
- [Manage API tokens for service accounts](https://support.atlassian.com/user-management/docs/manage-api-tokens-for-service-accounts/)
