# syntax=docker/dockerfile:1

# --- Build stage: compile the entrypoint binary ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod ./
# No go.sum needed (no external dependencies)

COPY entrypoint/ ./entrypoint/

RUN CGO_ENABLED=0 GOOS=linux go build -o /entrypoint ./entrypoint/

# --- Runtime stage: minimal image with agent tooling ---
FROM alpine:3.21

# Create the agent user and workspace directory.
RUN addgroup -g 1000 agent && \
    adduser -D -u 1000 -G agent -h /home/agent agent && \
    mkdir -p /workspace && \
    chown agent:agent /workspace

# Install common tools agents may need.
RUN apk add --no-cache \
    bash \
    curl \
    git \
    openssh-client \
    python3 \
    nodejs \
    npm

# Copy the entrypoint binary.
COPY --from=builder /entrypoint /usr/local/bin/entrypoint

WORKDIR /workspace

# The entrypoint starts as root (needed for privilege drop),
# reads control plane env vars, strips them, drops to the agent
# user, and execs the agent command.
ENTRYPOINT ["/usr/local/bin/entrypoint"]
