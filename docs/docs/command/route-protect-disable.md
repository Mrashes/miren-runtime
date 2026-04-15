---
title: "miren route protect disable"
sidebar_label: "route protect disable"
description: "Remove protection from an HTTP route"
---

# miren route protect disable

Remove protection from an HTTP route

:::note
This command requires the `routeoidc` [labs feature](/labs) to be enabled.
:::

## Usage

```bash
miren route protect disable <host> [flags]
```

## Arguments

- `host` — Hostname for the route (e.g., example.com)

## Flags

- `--cluster, -C` — Cluster name
- `--config` — Path to the config file
- `--default` — Remove protection from the default route

## Global Options

- `--options` — Path to file containing options
- `--server-address` — Server address to connect to (default: `127.0.0.1:8443`)
- `--verbose, -v` — Enable verbose output

## Examples

**Remove protection from a route:**

```bash
miren route protect disable example.com
```

## See also

- [`miren route protect`](/command/route-protect)
