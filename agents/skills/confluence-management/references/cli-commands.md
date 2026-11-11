# CLI Commands Reference

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--space` | string | from config | Override active space key |
| `--format` | string | json | Output format: json, compact, text |

## auth

Setup authentication.

Setup and cleanup change local authentication state; run them only when the
operator requests it. The examples use reserved `.example` placeholders.
Use the approved real URL for setup: a Cloud site has an `.atlassian.net` host
and `/wiki` path. The operator supplies credentials through the admitted secret
flow; agents must not copy real tokens into arguments or logs.

```bash
# Cloud
confluence-mgmt auth set-access --instance https://confluence-cloud.example/wiki --email user@company.example --token TOKEN

# Server/DC
confluence-mgmt auth set-access --instance https://confluence-server.example --token PAT

# Interactive
confluence-mgmt auth
```

Canonical validation and debug flow:

`version` is the executable readiness check. `auth whoami` performs a live
request and may refresh non-secret instance/auth-type configuration. `auth
clean` removes saved authentication; it is not a read-only diagnostic.

```bash
confluence-mgmt auth whoami
confluence-mgmt auth resolve
confluence-mgmt auth config-path
confluence-mgmt auth clean
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
