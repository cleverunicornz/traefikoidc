# Multi-stage build: prepare plugin with Go, then copy to Traefik
FROM golang:1.23-alpine AS plugin-builder

# Install git for go mod operations
RUN apk add --no-cache git

# Copy plugin source and prepare it
WORKDIR /plugins/src/github.com/cleverunicornz/traefikoidc
COPY . .

# Download dependencies
RUN go mod download
RUN go mod tidy

# Final stage: Use official Traefik with prepared plugin
FROM traefik:latest

# Copy the prepared plugin from builder stage
COPY --from=plugin-builder /plugins/src/github.com/cleverunicornz/traefikoidc /plugins/src/github.com/cleverunicornz/traefikoidc

# Traefik will use:
# --experimental.localplugins.traefikoidc.modulename=github.com/cleverunicornz/traefikoidc
# and load from /plugins/src/github.com/cleverunicornz/traefikoidc

EXPOSE 80 443 8080

ENTRYPOINT ["traefik"]
