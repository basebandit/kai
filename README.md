# Kubernetes MCP Server

A Model Context Protocol (MCP) server for managing a Kubernetes cluster through llms like Claude.

## Overview

This Kubernetes MCP server provides a bridge between large language models (LLMs) and Kubernetes clusters, enabling users to interact with their Kubernetes resources through natural language. The server exposes a comprehensive set of tools for managing clusters, namespaces, pods, deployments, services, and other Kubernetes resources.

## Features

<details>
<summary><b>Cluster Management</b></summary>

- [x] Connect to a Kubernetes cluster
- [ ] List all clusters in kubeconfig
- [ ] Switch between clusters
- [ ] Get cluster information
- [ ] Check cluster health
</details>

<details>
<summary><b>Namespace Operations</b></summary>

- [ ] List all namespaces
- [ ] Create a namespace
- [ ] Get namespace details
- [ ] Delete a namespace
- [ ] Update namespace labels/annotations
</details>

<details>
<summary><b>Pod Operations</b></summary>

- [x] Create pods
  <details>
  <summary>Details</summary>
  
  - [x] Basic pod creation with image and name
  - [x] Pod creation with environment variables
  - [x] Pod creation with volume mounts
  - [x] Pod creation with resource limits/requests
  - [x] Pod creation with node selectors
  </details>

- [x] List pods
  <details>
  <summary>Details</summary>
  
  - [x] List pods in current namespace
  - [x] List pods across all namespaces
  - [x] List pods with label selectors
  - [x] List pods with field selectors
  - [x] Limit number of pods listed
  </details>

- [x] Get pod details
  <details>
  <summary>Details</summary>
  
  - [x] Get basic pod information
  - [x] Get pod status
  - [x] Get pod IP and node information
  </details>

- [x] Delete pods
  <details>
  <summary>Details</summary>
  
  - [x] Delete pod by name
  - [x] Force delete pods
  </details>

- [x] Get logs from pods
  <details>
  <summary>Details</summary>
  
  - [x] Stream logs from a specific container
  - [x] Get logs with tail lines limit
  - [x] Get logs from previous container instance
  - [x] Get logs since a specific time
  </details>
</details>

<details>
<summary><b>Deployment Operations</b></summary>

- [x] Create deployments
  <details>
  <summary>Details</summary>
  
  - [x] Basic deployment with image and name
  - [x] Deployment with replica count
  - [x] Deployment with labels
  - [x] Deployment with environment variables
  - [x] Deployment with container ports
  - [x] Deployment with image pull policy/secrets
  </details>

- [x] List deployments
  <details>
  <summary>Details</summary>
  
  - [x] List deployments in current namespace
  - [x] List deployments across all namespaces
  - [x] List deployments with label selectors
  </details>

- [ ] Get deployment details
  <details>
  <summary>Details</summary>
  
  - [ ] Get basic deployment information
  - [ ] Get deployment status
  - [ ] Get deployment scaling information
  </details>

- [ ] Update deployments
  <details>
  <summary>Details</summary>
  
  - [ ] Scale deployments (change replica count)
  - [ ] Update deployment images
  - [ ] Update deployment configuration
  - [ ] Rollout restart
  </details>

- [ ] Delete deployments
  <details>
  <summary>Details</summary>
  
  - [ ] Delete deployment by name
  - [ ] Cascade delete (with or without dependent resources)
  </details>
</details>

<details>
<summary><b>Service Operations</b></summary>

- [ ] Create services
  <details>
  <summary>Details</summary>
  
  - [ ] Create ClusterIP service
  - [ ] Create NodePort service
  - [ ] Create LoadBalancer service
  - [ ] Create ExternalName service
  - [ ] Create headless service
  </details>

- [ ] List services
  <details>
  <summary>Details</summary>
  
  - [ ] List services in current namespace
  - [ ] List services across all namespaces
  - [ ] List services with label selectors
  </details>

- [ ] Get service details
  <details>
  <summary>Details</summary>
  
  - [ ] Get service endpoints
  - [ ] Get service ports
  - [ ] Get service selectors
  </details>

- [ ] Delete services
  <details>
  <summary>Details</summary>
  
  - [ ] Delete service by name
  </details>
</details>

<details>
<summary><b>Ingress Operations</b></summary>

- [ ] Create ingress
  <details>
  <summary>Details</summary>
  
  - [ ] Create basic ingress with host and path
  - [ ] Create ingress with TLS
  - [ ] Create ingress with annotations
  </details>

- [ ] List ingresses
  <details>
  <summary>Details</summary>
  
  - [ ] List ingresses in current namespace
  - [ ] List ingresses across all namespaces
  </details>

- [ ] Get ingress details
- [ ] Update ingress
- [ ] Delete ingress
</details>

<details>
<summary><b>ConfigMap & Secret Operations</b></summary>

- [ ] Create ConfigMaps
  <details>
  <summary>Details</summary>
  
  - [ ] Create from literal values
  - [ ] Create from files
  </details>

- [ ] List ConfigMaps
- [ ] Get ConfigMap details
- [ ] Update ConfigMaps
- [ ] Delete ConfigMaps
- [ ] Create Secrets
  <details>
  <summary>Details</summary>
  
  - [ ] Create generic secrets
  - [ ] Create TLS secrets
  - [ ] Create Docker registry secrets
  </details>

- [ ] List Secrets
- [ ] Get Secret details
- [ ] Update Secrets
- [ ] Delete Secrets
</details>

<details>
<summary><b>Job & CronJob Operations</b></summary>

- [ ] Create Jobs
  <details>
  <summary>Details</summary>
  
  - [ ] Create basic job
  - [ ] Create job with completion/parallelism
  </details>

- [ ] List Jobs
- [ ] Get Job details
- [ ] Delete Jobs
- [ ] Create CronJobs
  <details>
  <summary>Details</summary>
  
  - [ ] Create with schedule
  - [ ] Create with suspend option
  </details>

- [ ] List CronJobs
- [ ] Get CronJob details
- [ ] Delete CronJobs
</details>

<details>
<summary><b>Node Operations</b></summary>

- [ ] List all nodes
- [ ] Get node details
  <details>
  <summary>Details</summary>
  
  - [ ] Get node capacity/allocatable resources
  - [ ] Get node conditions
  - [ ] Get node labels/taints
  </details>

- [ ] Cordon/Uncordon nodes
- [ ] Drain nodes
</details>

<details>
<summary><b>Utility Operations</b></summary>

- [ ] Port forwarding
  <details>
  <summary>Details</summary>
  
  - [ ] Port forward to a pod
  - [ ] Port forward to a service
  </details>

- [ ] Get Kubernetes events
  <details>
  <summary>Details</summary>
  
  - [ ] Get cluster-wide events
  - [ ] Get namespace events
  - [ ] Get events for a specific resource
  </details>

- [ ] kubectl explain support
  <details>
  <summary>Details</summary>
  
  - [ ] Get resource documentation
  - [ ] Get field documentation
  </details>

- [ ] kubectl api-resources support
  <details>
  <summary>Details</summary>
  
  - [ ] List all available API resources
  - [ ] Get API resource details
  </details>
</details>

<details>
<summary><b>Persistent Volume Operations</b></summary>

- [ ] Create PersistentVolumes
- [ ] List PersistentVolumes
- [ ] Get PersistentVolume details
- [ ] Delete PersistentVolumes
- [ ] Create PersistentVolumeClaims
- [ ] List PersistentVolumeClaims
- [ ] Get PersistentVolumeClaim details
- [ ] Delete PersistentVolumeClaims
</details>

<details>
<summary><b>RBAC Operations</b></summary>

- [ ] Create/List/Get/Delete ServiceAccounts
- [ ] Create/List/Get/Delete Roles
- [ ] Create/List/Get/Delete RoleBindings
- [ ] Create/List/Get/Delete ClusterRoles
- [ ] Create/List/Get/Delete ClusterRoleBindings
</details>

<details>
<summary><b>Custom Resource Operations</b></summary>

- [ ] List custom resource definitions
- [ ] Get custom resource definition details
- [ ] List custom resources of a specific type
- [ ] Create/Get/Update/Delete custom resources
</details>

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