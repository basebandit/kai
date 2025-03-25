# Kubernetes MCP Server

A Model Context Protocol (MCP) server for managing Kubernetes resources through llms like Claude.

## Overview

This Kubernetes MCP server provides a bridge between large language models (LLMs) and Kubernetes clusters, enabling users to interact with their Kubernetes resources through natural language. The server exposes a comprehensive set of tools for managing clusters, namespaces, pods, deployments, services, and other Kubernetes resources.

## Features

- **Cluster Management**: Connect to multiple Kubernetes clusters and switch between contexts
- **Resource Operations**: Create, read, update, and delete Kubernetes resources
- **Pod Management**: List pods, get pod details, stream logs, and delete pods
- **Deployment Management**: Create and manage deployments across namespaces
- **Service Operations**: Interact with Kubernetes services
- **YAML Support**: Apply Kubernetes manifests directly from YAML
- **Custom Resource Support**: Work with custom resource definitions (CRDs)

## Installation

To install the Kubernetes MCP server, run:

```sh
go install github.com/basebandit/kai@latest
```

## Integration with Claude for Desktop

`code ~/Library/Application\ Support/Claude/claude_desktop_config.json`

Add the server to your **Claude for Desktop** configuration by editing `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "/path/to/kubernetes-mcp-server"
    }
  }
}
```


![Kubernetes MCP Server](./claude_desktop.png)