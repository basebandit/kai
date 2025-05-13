# Kubernetes MCP Server

A Model Context Protocol (MCP) server for managing a Kubernetes cluster through llms like Claude.

## Overview

This Kubernetes MCP server provides a bridge between large language models (LLMs) and Kubernetes clusters, enabling users to interact with their Kubernetes resources through natural language. The server exposes a comprehensive set of tools for managing clusters, namespaces, pods, deployments, services, and other Kubernetes resources.

## Features

- [x] Connect to a Kubernetes cluster
- [x] List all pods
- [x] List all deployments 
- [] List all services
- [] List all nodes
- [] Create, describe, delete a pod
- [] List all namespaces, create a namespace
- [] Create custom pod & deployment configs, 
- [] update deployment replicas
- [x] Get logs from a pod for debugging
- [ ] kubectl explain and kubectl api-resources support
- [ ] Get Kubernetes events from the cluster
- [ ] Port forward to a pod or service
- [ ] Create, list, and decribe cronjobs

## Requirements 
The server will by default connect to your current context. Make sure you have:  

Access to a Kubernetes cluster configured for kubectl (e.g. minikube, Rancher Desktop,  EKS, GKE, etc.)

## Installation

To install the Kubernetes MCP server, run:

```sh
go install github.com/basebandit/kai/cmd/kai
```

## Integration with Claude for Desktop

`code ~/Library/Application\ Support/Claude/claude_desktop_config.json`

Add the server to your **Claude for Desktop** configuration by editing `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "/path/to/kubernetes-mcp-server-binary"
    }
  }
}
```


![Kubernetes MCP Server](./claude_desktop.png)