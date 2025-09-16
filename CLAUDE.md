# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Core Build Commands

### Essential Commands
- `make build` - Build all go binaries for host architecture
- `make test` - Run all unit tests (includes racetest)
- `make lint` - Run all linters (go, python, copyright, scripts, etc.)
- `make format` - Auto-format all source code (required before PRs)
- `make clean` - Delete build artifacts
- `make docker` - Build containers in local docker cache

### Development Workflow
- `make precommit` - Runs format and lint (recommended before commits)
- `make DEBUG=1 build` - Build with debugger information
- `make coverage` - Generate unit test coverage reports
- `make racetest` - Run tests with race detection

### Testing Specific Components
- Run specific tests: `go test ./pilot/pkg/networking/core/ -v`
- Run benchmarks: `make benchtest`

## Project Structure

### Core Components
- **`pilot/`** - Service discovery, configuration management, and proxy configuration
  - `pilot-discovery` - Main control plane component (Istiod)
  - `pilot-agent` - Runs alongside workload proxies
- **`security/`** - Certificate Authority (Citadel) and security-related functionality
- **`istioctl/`** - Command-line tool for Istio configuration and debugging
- **`pkg/`** - Shared libraries and common functionality across components

### Key Directories
- **`manifests/`** - Helm charts and Kubernetes manifests
- **`samples/`** - Example configurations and applications
- **`tests/integration/`** - Integration test suites
- **`tools/`** - Build tools and development utilities

## Code Architecture

### Control Plane Components
- **Pilot** (`pilot/pkg/`) - Manages proxy configuration and service discovery
- **Security** (`security/pkg/`) - Handles certificate issuance and rotation
- **Configuration** (`pkg/config/`) - Manages Istio configuration resources

### Data Plane Integration
- **XDS Server** (`pilot/pkg/xds/`) - Serves configuration to Envoy proxies
- **Bootstrap** (`pkg/bootstrap/`) - Proxy bootstrap configuration generation
- **Networking** (`pilot/pkg/networking/`) - Traffic management policy translation

### Key Patterns
- Configuration is managed through Kubernetes CRDs
- Components use informer-based controllers for Kubernetes integration
- XDS (Envoy Discovery Service) protocol for proxy configuration distribution

## Environment Setup

Required environment variables for development:
```bash
export HUB="docker.io/$USER"
export TAG=$USER
```

Optional container-based builds:
```bash
export BUILD_WITH_CONTAINER=1  # Use containers for building
```

## Go-Specific Guidelines

### Code Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html) principles
- Use lowercase for package names, singular nouns for types
- Comment thoroughly, especially non-obvious implementation choices
- Use camelCase for Go types, lowercase with underscores for file names

### Performance Best Practices
- Preallocate and reuse memory to reduce GC pressure
- Use `sync.Pool` for object pooling when appropriate
- Avoid creating goroutines in the main request serving path
- Prefer inlined structs over pointer references when possible
- Design APIs to minimize memory allocations

### Testing Standards
- Achieve >98% unit test coverage
- Use table-driven tests where appropriate
- Write integration tests for significant features
- Avoid timing-sensitive test implementations

## Development Prerequisites

- Go 1.24.0+ (as specified in go.mod)
- Docker for container builds
- Make for build system
- Kubernetes 1.16+ for testing

## Integration Test Environment

Default configuration files:
- `tests/integration/iop-integration-test-defaults.yaml` - Standard test config
- `tests/integration/iop-ambient-test-defaults.yaml` - Ambient mesh config

Common test make targets are defined in `tests/integration/tests.mk`.