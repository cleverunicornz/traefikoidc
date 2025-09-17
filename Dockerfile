# Simple approach: Use Traefik with plugin copied directly
FROM traefik:latest

# Copy plugin source to the local plugins directory
# Traefik will load this when using --experimental.localplugins
COPY . /plugins/src/github.com/cleverunicornz/traefikoidc/

# Set working directory back to root
WORKDIR /

# Traefik will use:
# --experimental.localplugins.traefikoidc.modulename=github.com/cleverunicornz/traefikoidc
# and load from /plugins/src/github.com/cleverunicornz/traefikoidc

EXPOSE 80 443 8080

ENTRYPOINT ["traefik"]
