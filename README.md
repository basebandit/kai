<p align="center">
  <img src="./kai.jpg" alt="Kai Logo">
</p>

# Kai - Kubernetes MCP Server

A Model Context Protocol (MCP) server for managing Kubernetes clusters through LLM clients like Claude and Ollama.

## Overview

Kai provides a bridge between large language models (LLMs) and your Kubernetes clusters, enabling natural language interaction with Kubernetes resources. The server exposes a comprehensive set of tools for managing clusters, namespaces, pods, deployments, services, and other Kubernetes resources.

## Features

### Core Workloads
- [x] **Pods** - Create, list, get, delete, and stream logs
- [x] **Deployments** - Create, list, describe, and update
- [x] **Jobs** - Batch workload management (create, get, list, delete)
- [x] **CronJobs** - Scheduled batch workloads (create, get, list, delete)

### Networking
- [x] **Services** - Create, get, list, and delete
- [x] **Ingress** - HTTP/HTTPS routing, TLS configuration (create, get, list, update, delete)

### Configuration
- [x] **ConfigMaps** - Configuration management (create, get, list, update, delete)
- [x] **Secrets** - Secret management (create, get, list, update, delete)
- [x] **Namespaces** - Namespace management (create, get, list, delete)

### Cluster Operations
- [x] **Context Management** - Switch contexts, list contexts, rename, delete
- [ ] **Nodes** - Node monitoring, cordoning, and draining
- [ ] **Cluster Health** - Cluster status and resource metrics

### Storage
- [ ] **Persistent Volumes** - PV and PVC management
- [ ] **Storage Classes** - Storage class operations

### Security
- [ ] **RBAC** - Roles, RoleBindings, and ServiceAccounts

### Advanced
- [ ] **Custom Resources** - CRD and custom resource operations
- [ ] **Utilities** - Port forwarding, events, and API exploration

## Requirements

The server connects to your current kubectl context by default. Ensure you have access to a Kubernetes cluster configured for kubectl (e.g., minikube, Rancher Desktop, kind, EKS, GKE, AKS).

## Installation

```sh
go install github.com/basebandit/kai/cmd/kai@latest
```

## Configuration

### Claude for Desktop

Edit your Claude Desktop configuration:

```sh
# macOS
code ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Linux
code ~/.config/Claude/claude_desktop_config.json
```

Add the server configuration:

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "/path/to/kai"
    }
  }
}
```

### Custom Kubeconfig

By default, Kai uses `~/.kube/config`. The server automatically loads your current context on startup.

## Usage Examples

Once configured, you can interact with your cluster using natural language:

- "List all pods in the default namespace"
- "Create a deployment named nginx with 3 replicas using the nginx:latest image"
- "Show me the logs for pod my-app"
- "Delete the service named backend"
- "Create a cronjob that runs every 5 minutes"
- "Create an ingress for my-app with TLS enabled"

## Contributing

Contributions are welcome! Please see our contributing guidelines for more information.

## License

This project is licensed under the MIT License.

---

![Kubernetes MCP Server](./claude_desktop.png)