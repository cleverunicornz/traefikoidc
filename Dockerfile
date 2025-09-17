# Use standard Traefik with your plugin via local plugins
FROM traefik:latest

# Copy your plugin source into the container
COPY . /plugins/src/github.com/cleverunicornz/traefikoidc/

WORKDIR /plugins/src/github.com/cleverunicornz/traefikoidc

# Install git for module operations
RUN apk add --no-cache git

# Set the plugin as a local module
RUN go mod download

WORKDIR /

# Traefik will use --experimental.localplugins.traefikoidc.modulename=github.com/cleverunicornz/traefikoidc
# and load from /plugins/src/github.com/cleverunicornz/traefikoidc

EXPOSE 80 443 8080

ENTRYPOINT ["traefik"]
