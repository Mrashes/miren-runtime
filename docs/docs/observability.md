---
sidebar_position: 9
---

# Observability

Miren instruments the request lifecycle with [OpenTelemetry](https://opentelemetry.io/) distributed tracing. Traces flow from the initial HTTP request through internal routing, sandbox management, and container operations, giving you visibility into what's happening at every layer.

## What Miren Traces

Every HTTP request that arrives at Miren generates a trace with spans covering:

- **httpingress** — The full request lifecycle: routing, lease acquisition, and proxying to your app
- **httpingress.lease** — Sandbox lease management, including whether a cached lease was used or a cold start was required
- **RPC calls** — Internal service-to-service communication within Miren
- **containerd gRPC** — Container operations like image pulls, container creation, and task management

The most useful spans for app developers are `httpingress` (overall request latency) and `httpingress.lease` (cold start visibility). The RPC and containerd spans are primarily useful for operators debugging Miren itself.

## Trace Context Propagation

Miren participates in [W3C Trace Context](https://www.w3.org/TR/trace-context/) propagation in both directions:

**Inbound:** If your request includes a `traceparent` header, Miren continues that trace rather than starting a new one. This means requests from an instrumented frontend or upstream service produce a single connected trace that includes Miren's processing.

**Outbound:** When Miren forwards a request to your app, it injects a `traceparent` header. Your app can pick this up to create child spans that appear in the same trace as the Miren infrastructure spans.

## Connecting Your App's Traces

To participate in Miren's traces, configure your app with an OpenTelemetry SDK and point it at an OTLP-compatible backend. The SDK will automatically read the `traceparent` header from incoming requests and create child spans.

Set these environment variables on your app:

```toml
[[env]]
key = "OTEL_EXPORTER_OTLP_ENDPOINT"
value = "https://your-otel-collector:4318"

[[env]]
key = "OTEL_EXPORTER_OTLP_HEADERS"
value = "Authorization=Bearer your-api-key"

[[env]]
key = "OTEL_SERVICE_NAME"
value = "my-app"
```

This works with any OTel-compatible backend: Grafana Tempo, Honeycomb, Datadog, Jaeger, and others.

### Python Example

Using the OpenTelemetry auto-instrumentation for Flask:

```bash
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install
```

```toml
[[env]]
key = "OTEL_EXPORTER_OTLP_ENDPOINT"
value = "https://your-otel-collector:4318"

[[env]]
key = "OTEL_SERVICE_NAME"
value = "my-flask-app"
```

```text
# Procfile
web: opentelemetry-instrument flask run --host 0.0.0.0 --port 3000
```

The `opentelemetry-instrument` wrapper automatically reads the `traceparent` header from incoming requests and creates spans for your Flask routes.

### Node.js Example

Using the OpenTelemetry auto-instrumentation for Node.js:

```bash
npm install @opentelemetry/auto-instrumentations-node
```

```toml
[[env]]
key = "OTEL_EXPORTER_OTLP_ENDPOINT"
value = "https://your-otel-collector:4318"

[[env]]
key = "OTEL_SERVICE_NAME"
value = "my-node-app"

[[env]]
key = "NODE_OPTIONS"
value = "--require @opentelemetry/auto-instrumentations-node/register"
```

The `--require` flag loads the auto-instrumentation before your app starts, automatically instrumenting HTTP, Express, and other common libraries.

## What a Trace Looks Like

A typical request trace shows the full path through Miren:

```
httpingress                          [350ms]
├─ httpingress.lease                 [200ms]  (cold start)
│  ├─ rpc.call.AcquireLease         [195ms]
│  │  ├─ containerd...Images/Pull   [150ms]
│  │  └─ containerd...Tasks/Create  [40ms]
├─ [proxy to app]                   [150ms]
│  └─ my-app: GET /api/users        [145ms]  (your app's span)
```

On a warm request where a sandbox is already running:

```
httpingress                          [15ms]
├─ httpingress.lease                 [0.1ms]  (cached lease)
├─ [proxy to app]                   [14ms]
│  └─ my-app: GET /api/users        [12ms]
```

## Next Steps

- [Services](/services) — Configure your app's services
- [Application Scaling](/scaling) — Understand cold starts and autoscaling
