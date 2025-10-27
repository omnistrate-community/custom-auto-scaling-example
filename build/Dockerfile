# Stage 1: Build
FROM golang:1.25.3-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

# Set environment variables for build time
ENV TARGETOS=${TARGETOS}
ENV TARGETARCH=${TARGETARCH}

# Set the Current Working Directory inside the container
WORKDIR /go/src/controller

# Copy go.mod and go.sum before other files for dependency caching
COPY go.mod go.sum ./

RUN echo Fetching project dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

RUN echo Building and installing the controller service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build  \
      -o /go/bin/controller /go/src/controller

RUN ls -lrt /go/bin/proxyd

# Stage 2: Run
FROM alpine:latest AS final

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    apt-get install -y --no-install-recommends netcat-openbsd ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /go/bin/controller /bin/controller

# Expose port 
EXPOSE 3000

# Command to run the executable
ENTRYPOINT ["/bin/controller"]