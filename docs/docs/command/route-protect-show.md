---
title: "miren route protect show"
sidebar_label: "route protect show"
description: "Show protection for an HTTP route"
---

# miren route protect show

Show protection for an HTTP route

:::note
This command requires the `routeoidc` [labs feature](/labs) to be enabled.
:::

## Usage

```bash
miren route protect show <host> [flags]
```

## Arguments

- `host` — Hostname for the route (e.g., example.com)

## Flags

- `--cluster, -C` — Cluster name
- `--config` — Path to the config file
- `--default` — Show protection for the default route
- `--format` — Output format (text, json) (default: `text`)
- `--json` — Shorthand for --format json

## Global Options

- `--options` — Path to file containing options
- `--server-address` — Server address to connect to (default: `127.0.0.1:8443`)
- `--verbose, -v` — Enable verbose output

## Examples

**Show route protection:**

```bash
miren route protect show example.com
```

## See also

- [`miren route protect`](/command/route-protect)
