# Testing

This document describes how to set up and run the integration tests for the cert-manager external-dns webhook.

## Prerequisites

- Go 1.21+ installed
- Internet connection (for downloading test dependencies)

## Test Setup

The integration tests require Kubernetes test binaries (`etcd`, `kube-apiserver`, and `kubectl`) to run a local Kubernetes API server for testing.

### 1. Install setup-envtest

First, install the `setup-envtest` tool that manages the test dependencies:

```bash
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

### 2. Download Test Binaries

Download the required Kubernetes test binaries:

```bash
setup-envtest use -p env
```

This will output something like:
```
export KUBEBUILDER_ASSETS='/home/user/.local/share/kubebuilder-envtest/k8s/1.33.0-linux-amd64'
```

### 3. Set Environment Variables

The cert-manager test framework requires specific environment variables pointing to the test binaries. Run the following commands (adjust the path based on the output from step 2):

```bash
export KUBEBUILDER_ASSETS='/home/user/.local/share/kubebuilder-envtest/k8s/1.33.0-linux-amd64'
export TEST_ASSET_ETCD='/home/user/.local/share/kubebuilder-envtest/k8s/1.33.0-linux-amd64/etcd'
export TEST_ASSET_KUBE_APISERVER='/home/user/.local/share/kubebuilder-envtest/k8s/1.33.0-linux-amd64/kube-apiserver'
export TEST_ASSET_KUBECTL='/home/user/.local/share/kubebuilder-envtest/k8s/1.33.0-linux-amd64/kubectl'
```

### 4. Run Tests

Now you can run the integration tests:

```bash
go test -v ./...
```

## What the Tests Do

The integration tests:

1. **Start a mock DNS server** on port 59351 that simulates external-dns behavior
2. **Create a local Kubernetes API server** using the downloaded test binaries
3. **Install External-DNS CRDs** (DNSEndpoint resources) into the test cluster
4. **Run cert-manager ACME challenge tests** that:
   - Call the webhook's `Present()` method with a challenge request
   - Verify that a DNSEndpoint resource is created in Kubernetes
   - Simulate external-dns processing the DNSEndpoint to create DNS records
   - Verify that the DNS challenge can be resolved
   - Call the webhook's `CleanUp()` method to remove the challenge
   - Verify that the DNSEndpoint resource is deleted

## Test Structure

- **`main_test.go`** - Main integration test that uses cert-manager's ACME test suite
- **`testing/`** - Helper utilities for integration testing:
  - `solver.go` - Test wrapper for the webhook solver
  - `registry.go` - Mock external-dns registry that simulates DNS record management
  - `dnsendpoint.customresourcedefinitions.yaml` - External-DNS CRD definitions

## Troubleshooting

### "Failed to find integration test dependency" errors

Make sure you've properly set the environment variables pointing to the test binaries. The error message will indicate which specific binary is missing.

### Port conflicts

If port 59351 is already in use, the tests will fail. Make sure no other processes are using this port, or modify the `DNSPort` constant in `main_test.go`.

### Permission issues

The test binaries need to be executable. If you get permission denied errors, check that the binaries in your `KUBEBUILDER_ASSETS` directory have execute permissions.

## Automating Test Setup

You can create a script to set up the environment variables automatically:

```bash
#!/bin/bash
# test-setup.sh

eval $(setup-envtest use -p env)
export TEST_ASSET_ETCD="$KUBEBUILDER_ASSETS/etcd"
export TEST_ASSET_KUBE_APISERVER="$KUBEBUILDER_ASSETS/kube-apiserver"  
export TEST_ASSET_KUBECTL="$KUBEBUILDER_ASSETS/kubectl"

echo "Test environment configured. Run: go test -v ./..."
```

Then source it before running tests:
```bash
source test-setup.sh
go test -v ./...
```