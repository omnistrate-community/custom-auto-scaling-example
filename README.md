# Custom Auto Scaling Example for Omnistrate

A reference implementation demonstrating how to build custom autoscaling logic for resources deployed on the Omnistrate platform. This example shows how to create a standalone autoscaling controller that interacts with Omnistrate's local sidecar API to dynamically scale resources based on custom logic.

## Overview

This project provides a custom autoscaling controller written in Go that can scale Omnistrate resources up or down based on user-defined policies. Unlike standard autoscaling solutions that rely on predefined metrics (CPU, memory), this implementation allows you to define completely custom scaling logic while leveraging Omnistrate's capacity management API.

### Key Features

- **Custom Scaling Logic**: Define your own scaling policies and triggers
- **HTTP API**: Simple REST API for triggering scaling operations and checking status
- **Status Monitoring**: Real-time visibility into current capacity and scaling operations

## Architecture

The controller consists of several components:

1. **HTTP Server** (`cmd/controller.go`): Exposes REST endpoints for scaling operations and status checks
2. **Autoscaler** (`internal/autoscaler/`): Core scaling logic with cooldown management
3. **Omnistrate API Client** (`internal/omnistrate_api/`): Communicates with the Omnistrate sidecar
4. **Configuration** (`internal/config/`): Environment-based configuration management

### How It Works

1. The controller runs as a service alongside your Omnistrate resources
2. It communicates with the Omnistrate platform via a local sidecar API at `http://127.0.0.1:49750`
3. When a scaling request is received:
   - The controller checks if a scaling operation is already in progress
   - It waits for any active cooldown period to expire
   - It waits for the resource to be in an `ACTIVE` state
   - It gradually scales the resource to the target capacity in configured steps

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Docker (for containerized deployment)
- An Omnistrate service with autoscaling enabled

### Configuration

The controller is configured via environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AUTOSCALER_TARGET_RESOURCE` | Resource alias to scale (must match resource key in compose) | - | Yes |
| `AUTOSCALER_COOLDOWN` | Cooldown period in seconds between scaling operations | 300 | No |
| `AUTOSCALER_STEPS` | Number of capacity units to add/remove per operation | 1 | No |
| `AUTOSCALER_WAIT_FOR_ACTIVE_TIMEOUT` | Max time to wait for resource to become ACTIVE (seconds) | 900 | No |
| `AUTOSCALER_WAIT_FOR_ACTIVE_CHECK_INTERVAL` | Interval between status checks (seconds) | 30 | No |
| `DRY_RUN` | Enable dry-run mode (no actual API calls) | false | No |
| `PORT` | HTTP server port | 3000 | No |

### Example Service Configuration

Here's how to configure autoscaling in your `omnistrate-compose.yaml`:

```yaml
version: '3.9'

x-omnistrate-load-balancer:
  https:
    - name: frontend
      description: L7 Load Balancer for the controller UI
      paths:
        - associatedResourceKey: controller
          path: /
          backendPort: 3000

services:
  controller:
    depends_on:
      - worker
    image: ghcr.io/omnistrate-community/custom-auto-scaling-example:0.0.5
    ports:
      - '3000:3000'
    environment:
      - AUTOSCALER_COOLDOWN=300
      - AUTOSCALER_TARGET_RESOURCE=worker  # Must match the service name
      - AUTOSCALER_STEPS=1

  worker:
    x-omnistrate-mode-internal: true
    x-omnistrate-capabilities:
      autoscaling:
        policyType: custom       # Enable custom autoscaling
        maxReplicas: 6          # Maximum capacity
        minReplicas: 1          # Minimum capacity
    image: busybox:1.37.0
    command: ['sh', '-c', 'while true; do echo Working...; sleep 10; done']
```

**Key Configuration Notes:**

- Set `policyType: custom` in the `x-omnistrate-capabilities.autoscaling` section
- Set `x-omnistrate-mode-internal: true` to make the worker resource internal
- The `AUTOSCALER_TARGET_RESOURCE` must match the service key you want to scale (e.g., `worker`)
- The controller resource should have `depends_on` to ensure the target resource is created first

### Local Development

#### Build the Controller

```bash
make build
```

This creates a `controller` binary in the current directory.

#### Run Tests

```bash
make unit-test
```

## Scaling Behavior

### Cooldown Period

The cooldown period prevents rapid successive scaling operations:

- After each scaling action, the controller waits for the cooldown duration
- Default: 300 seconds (5 minutes)
- Configurable via `AUTOSCALER_COOLDOWN`

### Step-based Scaling

Scaling happens gradually in steps:

- Each operation adds or removes a fixed number of capacity units
- Default: 1 step per operation
- Configurable via `AUTOSCALER_STEPS`
- The controller will perform multiple operations if needed to reach the target

### State Management

The controller waits for the resource to be in an `ACTIVE` state before scaling:

- If a resource is `STARTING`, it waits until `ACTIVE`
- If a resource is `FAILED`, the operation fails
- Maximum wait time is configurable via `AUTOSCALER_WAIT_FOR_ACTIVE_TIMEOUT`

## Troubleshooting

### Controller Cannot Connect to Sidecar

**Problem**: Controller logs show connection errors to `http://127.0.0.1:49750`

**Solution**: Ensure the controller is deployed as part of an Omnistrate service. The sidecar is only available when running on Omnistrate.

## Contributing

Contributions are welcome! This is a reference implementation intended to demonstrate custom autoscaling patterns. Feel free to fork and adapt for your specific use case.

## License

See [LICENSE](LICENSE) file for details.

## Support

For issues related to:

- **This example**: Open an issue in this repository
- **Omnistrate platform**: Contact Omnistrate support or visit [omnistrate.com](https://omnistrate.com)
