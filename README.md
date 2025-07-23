# k8sless

A Kubernetes API adapter that manages Google Compute Engine VMs instead of Pods. It implements the Kubernetes Pod API but creates and manages GCE instances running Container-Optimized OS with kubelet.

## Overview

k8sless bridges the Kubernetes Pod API to Google Compute Engine, allowing you to:
- Create "pods" that are actually GCE VMs running Container-Optimized OS
- Run containers using kubelet in static pod mode
- Monitor pod status through the kubelet read-only API
- Collect logs and metrics in Google Cloud Logging and Monitoring

## Features

- **Pod to VM Translation**: Converts Kubernetes Pod specs to GCE instance configurations
- **Container Execution**: Uses kubelet to run containers on Container-Optimized OS
- **Real-time Status**: Queries kubelet API for accurate pod status
- **Cloud Logging**: Automatically collects container logs via fluent-bit
- **Cloud Monitoring**: Metrics collection enabled for observability
- **Auto-cleanup**: VMs are automatically deleted 1 minute after pod completion

## Usage

```bash
go run . --project=<your-gcp-project> --zone=<gce-zone>
```

This will:
1. Create a test pod that runs a simple container
2. Provision a GCE VM with Container-Optimized OS
3. Configure and start kubelet with the pod spec
4. Monitor the pod status until completion
5. Delete the VM after 1 minute

## Architecture

### How It Works

1. **Pod Creation**: When a pod is created via the Kubernetes API, k8sless:
   - Converts the Pod spec to a GCE Instance configuration
   - Stores the full Pod spec in instance metadata
   - Creates a VM with Container-Optimized OS (COS)
   - Uses cloud-init to configure and start kubelet

2. **Container Execution**: The VM boots and:
   - Retrieves the pod spec from metadata
   - Saves it to `/etc/kubernetes/manifests/`
   - Starts kubelet with static pod support
   - Kubelet reads the manifest and starts containers

3. **Status Monitoring**: k8sless watches the VM and:
   - Queries the kubelet read-only API on port 10255
   - Reports actual container status from kubelet
   - Falls back to VM status if kubelet isn't ready

4. **Logging & Monitoring**:
   - fluent-bit collects container logs from `/var/log/containers/`
   - Logs are sent to Google Cloud Logging with tag `k8sless_containers`
   - Metrics collection enabled via metadata flags

## Logging

Container logs are automatically collected and sent to Google Cloud Logging. 

### Viewing Logs

Use the Cloud Console or gcloud CLI:

```bash
gcloud logging read 'logName="projects/<project>/logs/k8sless_containers"' \
  --project=<project> --limit=10
```

### Log Format

Logs include:
- Container stdout/stderr output
- Timestamps
- Pod and container metadata
- k8sless=true label for filtering

## Configuration

### VM Configuration
- **Machine Type**: c4-standard-4 (hardcoded, TODO: make configurable)
- **OS**: Container-Optimized OS (latest stable)
- **Network**: Default VPC with external IP
- **Max Runtime**: 4 hours (configurable via Scheduling)

### Kubelet Configuration
- Static pod support enabled
- Read-only API on port 10255
- Anonymous authentication enabled
- No cluster DNS/domain configured

### Required Permissions

The default compute service account needs:
- `logging.write` - For sending logs
- `monitoring.write` - For sending metrics  
- `devstorage.read_only` - For pulling container images

## Limitations

- Only supports single-node pods (no multi-node orchestration)
- No service discovery or networking between pods
- No persistent volumes (only ephemeral storage)
- Requires `hostNetwork: true` (no CNI networking)
- Limited to Container-Optimized OS features

## Development

### Prerequisites
- Go 1.21+
- Google Cloud SDK (`gcloud`)
- GCP project with Compute Engine API enabled

### Building

```bash
go build -o k8sless .
```

### Project Structure
- `main.go` - CLI entry point and pod lifecycle management
- `client.go` - Kubernetes API client implementation
- `convert.go` - Pod to GCE instance conversion
- `watch.go` - Pod status monitoring via kubelet API
- `kubelet.go` - Kubelet API client
- `cloud-init.yaml` - VM initialization script (embedded)

## Future Improvements

- [ ] Support for resource limits and requests
- [ ] Configurable machine types based on pod resources
- [ ] Multi-container pod support
- [ ] Better error handling and retries
- [ ] Support for pod logs via K8s pod logs API
