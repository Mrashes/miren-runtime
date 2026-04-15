---
title: "miren route protect oidc"
sidebar_label: "route protect oidc"
description: "Protect an HTTP route with an OIDC identity provider"
---

# miren route protect oidc

Protect an HTTP route with an OIDC identity provider

:::note
This command requires the `routeoidc` [labs feature](/labs) to be enabled.
:::

## Usage

```bash
miren route protect oidc <host> [flags]
```

## Arguments

- `host` — Hostname for the route (e.g., example.com)

## Flags

- `--claim-header` — Claim to header mapping in format 'claim:header' (e.g., 'email:X-User-Email')
- `--client-id` — OAuth2 client ID (required with --provider-url)
- `--client-secret` — OAuth2 client secret (required with --provider-url)
- `--cluster, -C` — Cluster name
- `--config` — Path to the config file
- `--default` — Apply to the default route
- `--provider` — Name of existing identity provider (use --provider-url for inline creation)
- `--provider-url` — Identity provider URL (e.g., https://accounts.google.com) - creates provider if not exists
- `--scope` — OAuth2 scopes (can be specified multiple times)

## Global Options

- `--options` — Path to file containing options
- `--server-address` — Server address to connect to (default: `127.0.0.1:8443`)
- `--verbose, -v` — Enable verbose output

## Examples

**Protect a route with an existing OIDC provider:**

```bash
miren route protect oidc example.com --provider my-google-oidc
```

**Protect a route and create the OIDC provider inline:**

```bash
miren route protect oidc example.com \
  --provider-url https://accounts.google.com \
  --client-id my-client-id \
  --client-secret my-client-secret
```

## See also

- [`miren route protect`](/command/route-protect)
