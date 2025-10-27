# Omnistrate Custom Autoscaler

This autoscaler service reads configuration from environment variables and provides HTTP endpoints for scaling Omnistrate resources.

## Environment Variables

The autoscaler requires the following environment variables:

- `AUTOSCALER_COOLDOWN`: Cooldown period in seconds between scaling operations (default: 300)
- `AUTOSCALER_TARGET_RESOURCE`: Resource alias to scale (required)
- `INSTANCE_ID`: Instance ID to scale (required)

## Usage

### Starting the Controller

The main entry point is `cmd/controller.go`:

```bash
# Set environment variables
export AUTOSCALER_COOLDOWN=300
export AUTOSCALER_TARGET_RESOURCE="my-resource"
export INSTANCE_ID="instance-123"

# Run the controller
go run cmd/controller.go
```

Or build and run:

```bash
go build ./cmd/controller.go
./controller
```

### API Endpoints

The service exposes the following HTTP endpoints on port 8080 (configurable via `PORT` env var):

#### POST /scale
Scale the resource to a target capacity.

**Request:**
```json
{
  "targetCapacity": 5
}
```

**Response:**
```json
{
  "success": true,
  "message": "Successfully scaled to target capacity: 5"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/scale \
  -H "Content-Type: application/json" \
  -d '{"targetCapacity": 3}'
```

#### GET /status
Get current capacity and status of the resource.

**Response:**
```json
{
  "currentCapacity": 3,
  "status": "ACTIVE",
  "instanceId": "instance-123",
  "resourceAlias": "my-resource"
}
```

**Example:**
```bash
curl http://localhost:8080/status
```

#### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "service": "autoscaler"
}
```

## How It Works

1. **Environment Configuration**: The autoscaler loads configuration from environment variables on startup
2. **Cooldown Management**: Enforces a cooldown period between scaling operations to prevent rapid scaling
3. **Status Monitoring**: Waits for the instance to be in "ACTIVE" state before performing scaling operations
4. **Capacity Scaling**: Uses the Omnistrate API to get current capacity and add/remove capacity as needed

### Scaling Logic

When a scale request is received:

1. Check if we're within the cooldown period - if so, wait
2. Get current capacity using the Omnistrate API
3. Compare current capacity with target capacity
4. If scaling is needed:
   - Wait for instance to be in "ACTIVE" state
   - Add or remove capacity incrementally to reach target
   - Update last action time for cooldown tracking

### Status States

The autoscaler handles the following resource states:

- `ACTIVE`: Resource is ready for scaling operations
- `STARTING`: Resource is starting up - autoscaler will wait
- `PAUSED`: Resource is paused - autoscaler will wait
- `FAILED`: Resource is in failed state - scaling will fail
- `UNKNOWN`: Unknown state - autoscaler will wait

## Alternative Entry Points

### main.go
The project also includes a `main.go` file that provides the same functionality and can be used as an alternative entry point:

```bash
go run main.go
```

## Development

### Building
```bash
go build ./cmd/controller.go  # Build controller
go build .                    # Build main.go
```

### Testing
```bash
go test ./...
```

### Dependencies
The project uses:
- `github.com/go-openapi/strfmt` for timestamp handling
- `github.com/pkg/errors` for enhanced error handling

## Docker

To run in a container, you can use the provided Dockerfile:

```bash
docker build -t autoscaler .
docker run -p 8080:8080 \
  -e AUTOSCALER_COOLDOWN=300 \
  -e AUTOSCALER_TARGET_RESOURCE=my-resource \
  -e INSTANCE_ID=instance-123 \
  autoscaler
```