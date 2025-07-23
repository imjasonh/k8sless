# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

k8sless implements a Kubernetes API interface that uses Google Compute Engine (GCE) instances as the backing infrastructure instead of containers. It translates Kubernetes Pod operations into GCE VM operations, essentially treating VMs as "giant containers."

## Commands

**Build and Run:**
```bash
go build
go run . -project=<gcp-project> -zone=<gcp-zone>
```

**Development:**
```bash
go mod tidy     # Update dependencies
go test ./...   # Run tests (when added)
```

**Testing with firewall access:**
```bash
# Create firewall rule for kubelet API (one-time setup)
gcloud compute firewall-rules create k8sless-kubelet-api --project=<project> --allow=tcp:10255 --target-tags=kubelet-api --source-ranges=0.0.0.0/0

# Run and watch pod lifecycle
go run . -project=<project> -zone=<zone>

# Check kubelet API from external IP
curl http://<external-ip>:10255/pods | jq
```

## Architecture

The codebase follows an adapter pattern with several key components:

1. **Client Interface** (`client.go`): Implements `kubernetes.Interface`, routing Pod operations to GCE operations
2. **Conversion Layer** (`convert.go`): Bidirectional conversion between Kubernetes Pods and GCE Instances
   - Pod specs are serialized as JSON in instance metadata
   - Instance states map to Pod phases
   - Adds required TypeMeta fields for kubelet compatibility
3. **Watch Implementation** (`watch.go`): Queries kubelet API for real pod status instead of just VM status
4. **Kubelet Integration** (`kubelet.go`): Client for querying kubelet read-only API on port 10255
5. **Cloud-init Configuration** (`cloudinit.go`): Configures kubelet on VM startup

## How It Works

1. **Pod Creation**: When a pod is created via the Kubernetes API, k8sless:
   - Converts the Pod spec to a GCE Instance configuration
   - Stores the full Pod spec in instance metadata
   - Creates a VM with Container-Optimized OS (COS)
   - Cloud-init script configures and starts kubelet

2. **Kubelet Startup**: On VM boot, the cloud-init script:
   - Creates kubelet configuration with static pod support
   - Retrieves pod spec from metadata and saves to `/etc/kubernetes/manifests/pod.yaml`
   - Starts kubelet as a systemd service
   - Opens firewall ports for kubelet API access

3. **Status Monitoring**: The watcher:
   - Polls the kubelet API every 10 seconds
   - Reports actual pod status (Pending → Running → Succeeded/Failed)
   - Falls back to VM status if kubelet isn't ready

## Key Implementation Details

- **Supported Operations**: Create, Delete, Get, List, and Watch for Pods
- **Instance Configuration**: 
  - Uses Container-Optimized OS (COS) 
  - c4-standard-4 machine type (hardcoded in convert.go:58)
  - 4-hour max runtime
  - Network tags: "k8sless", "kubelet-api"
- **Metadata Storage**: 
  - `podspec`: Full Pod spec as JSON
  - `user-data`: Cloud-init configuration
- **Kubelet Configuration**:
  - Static pod path: `/etc/kubernetes/manifests`
  - Read-only API on port 10255
  - Anonymous auth enabled for simplicity
- **Networking**: 
  - Pods must use `hostNetwork: true` (no CNI configured)
  - RestartPolicy should be "Never" or "OnFailure"
- **Logging**: Uses chainguard-dev/clog for structured logging

## Important Notes

- **Container-Optimized OS Firewall**: COS has strict iptables rules. The cloud-init script opens ports 10255 (read-only API) and 10248 (health check)
- **Alternative approaches**: Instead of opening firewall, could use SSH tunneling or a sidecar container
- **Kubelet API Limitations**: The read-only API (10255) provides pod status but not logs. For logs, need:
  - Secure API on port 10250 (requires auth)
  - Direct file access via SSH
  - Google Cloud Logging integration (planned)

## Current Limitations

Most Pod operations return "not implemented" (see `not_implemented.go`), including:
- Updates, patches, status updates
- Logs (kubelet read-only API doesn't expose them)
- Exec, attach, port-forward operations
- Ephemeral containers, resizing

## Future Plans

- Configure logs and metrics collection to Google Cloud
- Update watcher to handle pod completion and trigger VM deletion
- Make machine type and other hardcoded values configurable
- Add support for resource limits and requests