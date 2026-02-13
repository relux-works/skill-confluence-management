# CLI Commands Reference

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--space` | string | from config | Override active space key |
| `--format` | string | json | Output format: json, compact, text |

## auth

Setup authentication.

```bash
# Cloud
confluence-mgmt auth --instance https://company.atlassian.net/wiki --email user@co.com --token TOKEN

# Server/DC
confluence-mgmt auth --instance https://confluence.company.com --token PAT

# Interactive
confluence-mgmt auth
```

## config

```bash
confluence-mgmt config show             # show config
confluence-mgmt config set space DEV    # set active space
```

## q (DSL query)

```bash
confluence-mgmt q '<query>'
```

See [DSL Examples](dsl-examples.md) for query syntax.

## page

```bash
confluence-mgmt page get 12345 --body              # get page with body
confluence-mgmt page create --space DEV --title T --body B --parent P --body-file F
confluence-mgmt page update 12345 --title T --body B --body-file F --message M
confluence-mgmt page delete 12345
```

## label

```bash
confluence-mgmt label add 12345 --labels "a,b,c"
confluence-mgmt label remove 12345 --labels "a,b"
```

## space

```bash
confluence-mgmt space list
```

## version

```bash
confluence-mgmt version
```
