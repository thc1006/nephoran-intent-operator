# Container Build Tools (No-Docker Environment)

This document describes the container image build toolchain installed on the
Nephoran development cluster. The cluster uses **containerd 2.2.1** as its
container runtime. Docker is intentionally absent.

## Installed Tools

| Tool | Version | Purpose |
|------|---------|---------|
| nerdctl | v2.2.1 | Docker-compatible CLI for containerd |
| buildkitd | v0.26.3 | Container image build daemon (OCI) |
| buildctl | v0.26.3 | BuildKit client CLI |
| buildah | v1.23.1 | Daemonless OCI image builder |

## Architecture

```
  nerdctl build        buildah bud
       |                    |
       v                    v
   buildkitd           buildah (daemonless)
       |                    |
       v                    v
   containerd          local OCI store
       |
       v
  /run/containerd/containerd.sock
       |
       v
  Kubernetes (k8s.io namespace)
```

nerdctl delegates builds to BuildKit (buildkitd), which stores images in
containerd. buildah operates independently with its own local store.

For Kubernetes pod usage, images must be in containerd's `k8s.io` namespace.

## Usage

### Building Images with nerdctl (Recommended)

nerdctl is a drop-in replacement for the `docker` CLI. All commands require
`sudo` because containerd runs as a system service.

```bash
# Build an image
sudo nerdctl build -t myapp:v1.0 -f Dockerfile .

# List images
sudo nerdctl images

# Run a container
sudo nerdctl run --rm myapp:v1.0

# Push to a registry
sudo nerdctl push registry.example.com/myapp:v1.0

# Tag an image
sudo nerdctl tag myapp:v1.0 registry.example.com/myapp:v1.0
```

### Building Images with buildah (Alternative)

buildah is daemonless and useful for CI pipelines or environments where
BuildKit is not available.

```bash
# Build from Dockerfile
sudo buildah bud -t myapp:v1.0 -f Dockerfile .

# List images
sudo buildah images

# Push to registry
sudo buildah push myapp:v1.0 docker://registry.example.com/myapp:v1.0
```

### Making Images Available to Kubernetes

Images built with `nerdctl` in the default namespace are NOT automatically
visible to Kubernetes. Use one of these methods:

**Method 1: Export and import (recommended)**

```bash
# Export from default namespace
sudo ctr -n default images export /tmp/myapp.tar docker.io/library/myapp:v1.0

# Import to k8s.io namespace
sudo ctr -n k8s.io images import /tmp/myapp.tar

# Clean up
rm /tmp/myapp.tar
```

**Method 2: Build directly in k8s.io namespace**

```bash
sudo nerdctl --namespace k8s.io build -t myapp:v1.0 -f Dockerfile .
```

**Method 3: Use in Kubernetes with imagePullPolicy: Never**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: myapp
    image: myapp:v1.0
    imagePullPolicy: Never
```

### BuildKit Direct Usage

For advanced build scenarios (multi-platform, cache export, etc.):

```bash
# Check BuildKit worker status
buildctl --addr unix:///run/buildkit/buildkitd.sock debug workers

# Build with cache export
buildctl --addr unix:///run/buildkit/buildkitd.sock build \
  --frontend dockerfile.v0 \
  --local context=. \
  --local dockerfile=. \
  --output type=image,name=myapp:v1.0
```

## Service Management

BuildKit runs as a systemd service:

```bash
# Check status
sudo systemctl status buildkit

# Restart
sudo systemctl restart buildkit

# View logs
sudo journalctl -u buildkit -f
```

## Troubleshooting

### "permission denied" when running nerdctl

nerdctl requires `sudo` for system-level containerd access:

```bash
sudo nerdctl ps    # correct
nerdctl ps         # will fail with permission error
```

### Image not found by Kubernetes

Ensure the image is in the `k8s.io` namespace:

```bash
sudo ctr -n k8s.io images ls | grep myapp
```

If not listed, export/import as described above.

### BuildKit connection errors

Verify the BuildKit daemon is running:

```bash
sudo systemctl status buildkit
sudo ls -la /run/buildkit/buildkitd.sock
```

### "rootless containerd" warning

This warning is informational. The cluster runs containerd in system mode
(not rootless). The warning can be safely ignored.

## Installation Details

Installed from the `nerdctl-full-2.2.1-linux-amd64.tar.gz` archive:
- Source: https://github.com/containerd/nerdctl/releases/tag/v2.2.1
- Binaries extracted to: `/usr/local/bin/`
- BuildKit service file: `/usr/local/lib/systemd/system/buildkit.service`
- buildah installed via: `apt-get install buildah`

## Verification Commands

```bash
# Verify all tools
nerdctl version
buildctl --version
buildkitd --version
buildah version

# Verify BuildKit daemon
sudo systemctl status buildkit

# Verify containerd socket
ls -la /run/containerd/containerd.sock

# End-to-end build test
echo 'FROM alpine:latest
CMD ["echo", "test"]' > /tmp/test.Dockerfile
sudo nerdctl build -t build-test:latest -f /tmp/test.Dockerfile /tmp
sudo nerdctl run --rm build-test:latest
sudo nerdctl rmi build-test:latest
rm /tmp/test.Dockerfile
```
