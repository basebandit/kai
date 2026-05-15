<p align="center">
  <img src="./kai.jpg" alt="Kai Logo" width="200">
</p>

<h1 align="center">Kai</h1>

<p align="center">
  <strong>Talk to your Kubernetes cluster using natural language</strong>
</p>

<p align="center">
  <a href="https://github.com/basebandit/kai/actions"><img src="https://github.com/basebandit/kai/workflows/CI/badge.svg" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/basebandit/kai"><img src="https://goreportcard.com/badge/github.com/basebandit/kai" alt="Go Report Card"></a>
  <a href="https://github.com/basebandit/kai/releases"><img src="https://img.shields.io/github/v/release/basebandit/kai" alt="Release"></a>
  <a href="https://github.com/basebandit/kai/blob/main/LICENSE"><img src="https://img.shields.io/github/license/basebandit/kai" alt="License"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#installation">Installation</a> •
  <a href="#what-can-i-do">What Can I Do?</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#production-deployment">Production</a>
</p>

---

Kai is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that lets you manage Kubernetes clusters through AI assistants like Claude, Cursor, and VS Code Copilot.

Instead of memorizing kubectl commands, just ask:

> "List all pods in the production namespace"
> "Scale the api deployment to 5 replicas"
> "Show me the logs for the failing pod"

---

## Quick Start

### 1. Install Kai

```bash
go install github.com/basebandit/kai/cmd/kai@latest
```

### 2. Add to Claude Desktop

Edit your config file:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kai"
    }
  }
}
```

### 3. Restart Claude Desktop

That's it! Start asking questions about your cluster.

---

## Installation

### Using Go (Recommended)

```bash
go install github.com/basebandit/kai/cmd/kai@latest
```

### Download Binary

Download from the [releases page](https://github.com/basebandit/kai/releases):

**macOS (Apple Silicon)**
```bash
curl -LO https://github.com/basebandit/kai/releases/latest/download/kai_Darwin_arm64.tar.gz
tar -xzf kai_Darwin_arm64.tar.gz
sudo mv kai /usr/local/bin/
```

**macOS (Intel)**
```bash
curl -LO https://github.com/basebandit/kai/releases/latest/download/kai_Darwin_x86_64.tar.gz
tar -xzf kai_Darwin_x86_64.tar.gz
sudo mv kai /usr/local/bin/
```

**Linux**
```bash
curl -LO https://github.com/basebandit/kai/releases/latest/download/kai_Linux_x86_64.tar.gz
tar -xzf kai_Linux_x86_64.tar.gz
sudo mv kai /usr/local/bin/
```

**Windows (PowerShell)**
```powershell
Invoke-WebRequest -Uri https://github.com/basebandit/kai/releases/latest/download/kai_Windows_x86_64.zip -OutFile kai.zip
Expand-Archive kai.zip -DestinationPath .
Move-Item kai.exe C:\Windows\System32\
```

### Build from Source

```bash
git clone https://github.com/basebandit/kai.git
cd kai
go build -o kai ./cmd/kai
sudo mv kai /usr/local/bin/
```

### Verify Installation

```bash
kai -version
```

---

## What Can I Do?

Here are some things you can ask your AI assistant once Kai is configured:

### Managing Pods
- "List all pods in the default namespace"
- "Show me pods that aren't running"
- "Get the logs from pod nginx-abc123"
- "Delete the crashed pod in staging"

### Working with Deployments
- "Create a deployment named api with 3 replicas using nginx:latest"
- "Scale the frontend deployment to 10 replicas"
- "Roll back the api deployment"
- "What's the status of all deployments?"

### Services & Networking
- "Create a service for the nginx deployment"
- "List all services in the production namespace"
- "Set up an ingress for api.example.com"

### Configuration
- "Create a configmap from these key-value pairs"
- "Show me all secrets in the default namespace"
- "Update the database configmap"

### Jobs & Scheduled Tasks
- "Create a job that runs the backup script"
- "Show me all cronjobs"
- "Suspend the nightly-cleanup cronjob"

### Debugging
- "Port forward the postgres service to localhost:5432"
- "Stream logs from all api pods"
- "Why is my pod failing?"

---

## Configuration

### Supported MCP Clients

<details>
<summary><strong>Claude Desktop</strong></summary>

Edit your config file:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kai"
    }
  }
}
```
</details>

<details>
<summary><strong>Claude Code (VS Code Extension)</strong></summary>

Run in terminal:
```bash
claude mcp add kubernetes -- kai
```

Or edit `~/.claude/settings.json`:
```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kai"
    }
  }
}
```
</details>

<details>
<summary><strong>Cursor</strong></summary>

Add to Cursor's MCP settings (Settings → MCP):

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kai"
    }
  }
}
```
</details>

<details>
<summary><strong>VS Code (GitHub Copilot)</strong></summary>

Add to your VS Code `settings.json`:

```json
{
  "mcp": {
    "servers": {
      "kubernetes": {
        "command": "kai"
      }
    }
  }
}
```
</details>

<details>
<summary><strong>Continue</strong></summary>

Add to `~/.continue/config.json`:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "kai"
        }
      }
    ]
  }
}
```
</details>

### Custom Kubeconfig

By default, Kai uses `~/.kube/config`. To use a different config:

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kai",
      "args": ["-kubeconfig", "/path/to/kubeconfig"]
    }
  }
}
```

### Multiple Clusters

Set up different kai instances for each cluster in Claude Desktop:

```json
{
  "mcpServers": {
    "k8s-prod": {
      "command": "kai",
      "args": ["-kubeconfig", "/Users/you/.kube/prod-config", "-context", "production"]
    },
    "k8s-staging": {
      "command": "kai",
      "args": ["-kubeconfig", "/Users/you/.kube/staging-config", "-context", "staging"]
    }
  }
}
```

> **Note**: Use absolute paths (not `~`) for kubeconfig files in MCP configurations.

---

## CLI Options

```
kai [options]

Options:
  -kubeconfig string      Path to kubeconfig file (default "~/.kube/config")
  -context string         Context name (default "local")
  -transport string       Transport mode: stdio or sse (default "stdio")
  -sse-addr string        SSE server address (default ":8080")
  -tls-cert string        TLS certificate file (for HTTPS)
  -tls-key string         TLS private key file (for HTTPS)
  -request-timeout        API request timeout (default 30s)
  -metrics                Enable Prometheus metrics (default true)
  -log-format string      Log format: json or text (default "json")
  -log-level string       Log level: debug, info, warn, error (default "info")
  -version                Show version
```

---

## Supported Resources

| Category | Resources | Operations |
|----------|-----------|------------|
| **Workloads** | Pods, Deployments, Jobs, CronJobs | Create, Read, Update, Delete, Scale, Logs |
| **Networking** | Services, Ingress | Create, Read, Update, Delete |
| **Config** | ConfigMaps, Secrets, Namespaces | Create, Read, Update, Delete |
| **Operations** | Contexts, Port Forwarding | Switch, List, Forward |

---

## Production Deployment

For production use cases, Kai supports SSE transport with TLS, health endpoints, and Prometheus metrics.

### SSE Mode (for web clients)

```bash
kai -transport=sse -sse-addr=:8080
```

With TLS:
```bash
kai -transport=sse -sse-addr=:8443 -tls-cert=cert.pem -tls-key=key.pem
```

### Health Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /healthz` | Liveness probe |
| `GET /readyz` | Readiness probe |
| `GET /metrics` | Prometheus metrics |

### Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `kai_requests_total` | Counter | Total requests by tool and status |
| `kai_request_duration_seconds` | Histogram | Request latency |
| `kai_active_connections` | Gauge | Active SSE connections |

### Kubernetes Deployment

<details>
<summary>Click to expand Kubernetes manifests</summary>

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kai
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kai
  template:
    metadata:
      labels:
        app: kai
    spec:
      serviceAccountName: kai
      containers:
        - name: kai
          image: ghcr.io/basebandit/kai:latest
          args: ["-transport=sse", "-sse-addr=:8080"]
          ports:
            - containerPort: 8080
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kai
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kai
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin  # Scope down for production!
subjects:
  - kind: ServiceAccount
    name: kai
    namespace: default
```
</details>

### Docker

```bash
docker run -v ~/.kube/config:/root/.kube/config:ro \
  ghcr.io/basebandit/kai:latest -transport=sse
```

---

## Viewing Logs

Kai outputs structured JSON logs to stderr.

**Claude Desktop logs location:**
- macOS: `~/Library/Logs/Claude/mcp-server-kubernetes.log`
- Linux: `~/.config/Claude/logs/mcp-server-kubernetes.log`

```bash
# Watch logs in real-time
tail -f ~/Library/Logs/Claude/mcp-server-kubernetes.log
```

---

## Troubleshooting

**Kai command not found**
```bash
# Check if kai is in your PATH
which kai

# If using go install, ensure GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

**Can't connect to cluster**
```bash
# Verify kubectl works
kubectl cluster-info
kubectl get nodes
```

**MCP client not seeing Kai**
1. Restart the MCP client after config changes
2. Check the config file syntax (valid JSON)
3. Verify the kai path is correct

**Enable debug logging**
```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kai",
      "args": ["-log-level", "debug"]
    }
  }
}
```

---

## Contributing

Contributions are welcome! See our [contributing guidelines](CONTRIBUTING.md).

```bash
git clone https://github.com/basebandit/kai.git
cd kai
go test ./...
go build -o kai ./cmd/kai
```

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <img src="./claude_desktop.png" alt="Kai in Claude Desktop" width="800">
</p>

<p align="center">
  <a href="https://mcpservers.org">Listed on MCP Servers</a>
</p>
