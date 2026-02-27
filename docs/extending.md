# Extending the Sandbox Image

The default image ships with a minimal set of tools (bash, curl, git, python3, nodejs, npm). Most agents will need more. This document covers how to customize the image for your use case.

## Adding system packages

Create a Dockerfile that extends the base image:

```dockerfile
FROM ghcr.io/travbz/sandbox-image:latest

# Install additional tools
RUN apk add --no-cache \
    jq \
    ripgrep \
    tmux \
    make \
    gcc \
    musl-dev
```

Build it:

```bash
docker build -t my-sandbox-image .
```

Then reference it in your `sandbox.toml`:

```toml
[sandbox]
image = "my-sandbox-image:latest"
```

## Adding Python packages

For Python-based agents:

```dockerfile
FROM ghcr.io/travbz/sandbox-image:latest

RUN pip3 install --break-system-packages \
    anthropic \
    openai \
    requests \
    pydantic
```

Or use a requirements file:

```dockerfile
FROM ghcr.io/travbz/sandbox-image:latest

COPY requirements.txt /tmp/requirements.txt
RUN pip3 install --break-system-packages -r /tmp/requirements.txt && \
    rm /tmp/requirements.txt
```

## Pre-installing an agent binary

If your agent is a compiled binary:

```dockerfile
FROM ghcr.io/travbz/sandbox-image:latest

COPY my-agent /usr/local/bin/my-agent
RUN chmod +x /usr/local/bin/my-agent
```

Then in `sandbox.toml`:

```toml
[sandbox]
image   = "my-sandbox-image:latest"
command = "my-agent"
args    = ["--verbose"]
```

## Changing the base image

The default base is `alpine:3.21`. If you need a different distro (e.g., Debian for better package compatibility):

```dockerfile
# --- Build stage (same) ---
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod ./
COPY entrypoint/ ./entrypoint/
RUN CGO_ENABLED=0 GOOS=linux go build -o /entrypoint ./entrypoint/

# --- Runtime stage (different base) ---
FROM debian:bookworm-slim

RUN groupadd -g 1000 agent && \
    useradd -m -u 1000 -g agent agent && \
    mkdir -p /workspace && \
    chown agent:agent /workspace

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash curl git openssh-client python3 nodejs npm \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /entrypoint /usr/local/bin/entrypoint
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/entrypoint"]
```

The entrypoint binary is statically compiled, so it works on any Linux distro. Just make sure:

1. The `agent` user exists in `/etc/passwd` (the entrypoint looks it up there).
2. `/workspace` exists and is owned by the agent user.

## Adding shared directory mount points

If your agent needs additional mount points beyond `/workspace`:

```dockerfile
FROM ghcr.io/travbz/sandbox-image:latest

RUN mkdir -p /data /output /cache && \
    chown agent:agent /data /output /cache
```

Then in `sandbox.toml`:

```toml
[shared_dirs]
workspace = { host = "./workspace", guest = "/workspace" }
data      = { host = "./data",      guest = "/data"      }
output    = { host = "./output",    guest = "/output"    }
cache     = { host = "./.cache",    guest = "/cache"     }
```

## Multi-arch builds

If you're extending the image and need it to run on both x86 and ARM (e.g., Raspberry Pi):

```bash
docker buildx create --name multiarch --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t my-sandbox-image:latest \
  --push .
```

The base image is already multi-arch, so your extended image will be too as long as you don't include architecture-specific binaries.

## Things to keep in mind

- **Don't override ENTRYPOINT.** The entrypoint binary handles env stripping and privilege drop. If you need a setup step before the agent, use a wrapper script as the `AGENT_COMMAND` instead.
- **Don't change the agent user's UID/GID** unless you also update your shared directory permissions. The default is 1000:1000.
- **Don't install secrets into the image.** Use `sandbox.toml` to inject them at runtime. Baked-in secrets leak when the image is shared.
- **Keep the image small.** Every MB matters when pulling on a Raspberry Pi over WiFi. Use `--no-cache` with apk and clean up temp files.
