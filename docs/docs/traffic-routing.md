---
sidebar_position: 7
---

# Traffic Routing

Miren routes traffic to your application through two mechanisms depending on the type of service:

- **HTTP services** are routed at Layer 7 through Miren's HTTP ingress (reverse proxy)
- **TCP/UDP services** are routed at Layer 4 through nftables NAT rules

Most apps only need HTTP routing, which works automatically. TCP/UDP routing is for services like databases, game servers, IRC, or anything that speaks a non-HTTP protocol.

## HTTP Routing

HTTP routing is the default and requires no special configuration. When you deploy an app with a `web` service, Miren's HTTP ingress handles TLS termination, hostname-based routing, and reverse proxying.

### How It Works

1. A request arrives at the node on port 80 or 443
2. The HTTP ingress extracts the hostname and looks up the matching route
3. The activator finds (or starts) a sandbox running the `web` service
4. The ingress reverse-proxies the request to the sandbox's HTTP port

```
Client → :443 → HTTP Ingress → Route Lookup → Activator → Sandbox :3000
```

### Routes

Routes map hostnames to apps. Create one with the CLI:

```bash
miren route add myapp.example.com --app myapp
```

Each hostname routes to the app's `web` service. TLS certificates are provisioned automatically via Let's Encrypt (see [TLS Certificates](/tls)).

### The `web` Service

The `web` service is special:

- It's the only service that receives HTTP traffic from the ingress
- If no port is configured, it defaults to port 3000
- The `PORT` environment variable is set automatically so your app knows which port to listen on
- It uses `auto` scaling by default (scale-to-zero when idle, scale up on traffic)

```toml
[services.web]
command = "node server.js"
# Listens on PORT=3000 by default
```

To use a different port:

```toml
[services.web]
command = "gunicorn app:app --bind 0.0.0.0:8000"
port = 8000
```

### The `PORT` Environment Variable

Miren sets `PORT` automatically based on your port configuration:

| Configuration | `PORT` value |
|---------------|-------------|
| No port configured (web service) | `3000` |
| Scalar `port = 8000` | `8000` |
| `ports[]` with an `http`-typed port | First `http`-typed port |
| `ports[]` with no `http`-typed port | First port in the array |

`PORT` is a system-managed variable and cannot be overridden by user env config.

## TCP/UDP Routing

For non-HTTP services — databases, game servers, IRC, gRPC without HTTP/2, raw TCP/UDP protocols — Miren routes traffic at Layer 4 using nftables NAT rules.

### Configuring Ports

Use the `ports` array in `app.toml` to expose non-HTTP ports:

```toml
[services.irc]
command = "./ircd"

[[services.irc.ports]]
port = 6667
name = "irc"
type = "tcp"

[[services.irc.ports]]
port = 6697
name = "irc-tls"
type = "tcp"
node_port = 6697
```

Each port entry has these fields:

| Field | Required | Description | Default |
|-------|----------|-------------|---------|
| `port` | Yes | Port the process listens on inside the container (1–65535) | — |
| `name` | Yes | Unique name for this port | — |
| `type` | No | `"http"` or `"tcp"` | `"http"` |
| `protocol` | No | `"tcp"` or `"udp"` | `"tcp"` |
| `node_port` | No | Port to expose on the host (0–65535) | (none) |

### Port Types

The `type` field determines how traffic is routed:

- **`http`** — Routed through the HTTP ingress (L7 reverse proxy). Only meaningful on the `web` service.
- **`tcp`** — Routed through nftables NAT rules (L4). Requires a `node_port` for external access.

### How L4 Routing Works

When a service has non-HTTP ports, Miren creates a Service entity that triggers the L4 routing pipeline:

1. **IP allocation** — The service gets a cluster-internal IP from the service prefix pool
2. **nftables rules** — The ServiceController programs NAT rules:
   - A service chain that load-balances across sandbox endpoints
   - Endpoint chains that DNAT traffic to individual sandbox IPs
   - NodePort rules that forward host-port traffic to the service chain
3. **Endpoints** — As sandboxes start and stop, endpoint entries are updated and nftables rules are reprogrammed

```
Client → :6697 (node_port) → nftables PREROUTING → Service Chain
    → Load Balance → Endpoint Chain → DNAT → Sandbox :6697
```

### NodePorts

A `node_port` exposes a service port directly on the host machine. This is how external clients reach non-HTTP services.

```toml
[[services.game.ports]]
port = 27015
name = "game"
type = "tcp"
protocol = "udp"
node_port = 27015
```

NodePort constraints:
- Must be unique across all apps on the cluster — Miren validates this at deploy time
- Cannot conflict with ports used by Miren itself (80, 443, 8443)

Without a `node_port`, L4 service ports are only reachable from within the cluster (useful for internal services like databases).

## Multi-Port Services

A single service can expose multiple ports with different types and protocols:

```toml
[services.app]
command = "./server"

# HTTP health/API endpoint — routed through HTTP ingress
[[services.app.ports]]
port = 3000
name = "http"
type = "http"

# TCP data port — routed through nftables
[[services.app.ports]]
port = 7000
name = "data"
type = "tcp"
node_port = 7000
```

This is common for services that need both an HTTP endpoint (for health checks, metrics, or API) and a raw protocol port (for data transfer).

### DNS Service (TCP + UDP on same port)

Some protocols like DNS require the same port on both TCP and UDP:

```toml
[services.dns]
command = "./dns-server"

[[services.dns.ports]]
port = 53
name = "dns-udp"
type = "tcp"
protocol = "udp"
node_port = 53

[[services.dns.ports]]
port = 53
name = "dns-tcp"
type = "tcp"
protocol = "tcp"
node_port = 5353
```

The same container port can be used for different protocols. Each `(port, protocol)` combination must be unique within a service.

## Internal Service Communication

Services within the same app communicate over the internal network using DNS names. Every service is reachable at `<service>.app.miren`:

```toml
name = "myapp"

[[env]]
key = "DATABASE_URL"
value = "postgres://user:pass@postgres.app.miren:5432/mydb"

[services.web]
command = "node server.js"

[services.postgres]
image = "postgres:16"

[services.postgres.concurrency]
mode = "fixed"
num_instances = 1
```

Internal communication uses direct container-to-container networking through the bridge — no NAT rules or ingress are involved. Services connect using standard ports (5432 for PostgreSQL, 6379 for Redis, etc.) without any Miren port configuration.

Internal DNS is for services **within the same app**. Cross-app communication is not currently supported.

## Port Configuration Reference

### Scalar Fields (backward compatible)

For simple single-port services, you can use the scalar fields:

```toml
[services.web]
port = 8000
port_name = "http"
port_type = "http"
```

| Field | Description | Default |
|-------|-------------|---------|
| `port` | Port the process listens on | `3000` (web only) |
| `port_name` | Name for the port | Service name |
| `port_type` | `"http"` or `"tcp"` | `"http"` |

### `ports[]` Array (multi-port)

For services with multiple ports or non-HTTP protocols, use the `ports` array:

```toml
[[services.myservice.ports]]
port = 3000
name = "http"
type = "http"

[[services.myservice.ports]]
port = 7000
name = "grpc"
type = "tcp"
protocol = "tcp"
node_port = 7000
```

You cannot mix scalar port fields and the `ports[]` array on the same service — Miren rejects this at deploy time.

## Examples

### IRC Server

```toml
name = "irc"

[services.irc]
command = "./ircd"

[[services.irc.ports]]
port = 6667
name = "irc"
type = "tcp"
node_port = 6667

[[services.irc.ports]]
port = 6697
name = "irc-tls"
type = "tcp"
node_port = 6697
```

### Game Server with HTTP Admin Panel

```toml
name = "gameserver"

[services.game]
command = "./server"

[[services.game.ports]]
port = 3000
name = "admin"
type = "http"

[[services.game.ports]]
port = 27015
name = "game"
type = "tcp"
protocol = "udp"
node_port = 27015

[services.game.concurrency]
mode = "fixed"
num_instances = 1
```

### TCP Echo Server (testing multi-port)

```toml
name = "tcp-echo"

[services.echo]
command = "./tcp-echo"

[[services.echo.ports]]
port = 3000
name = "health"
type = "http"

[[services.echo.ports]]
port = 7000
name = "echo"
type = "tcp"
node_port = 7000

[services.echo.concurrency]
mode = "fixed"
num_instances = 1
```

## Next Steps

- [Services](/services) — Defining services, commands, images, and scaling
- [TLS Certificates](/tls) — How HTTPS works for HTTP services
- [Firewall Configuration](/firewall) — Host-level firewall rules and cloud provider setup
- [Application Scaling](/scaling) — How services scale up and down
