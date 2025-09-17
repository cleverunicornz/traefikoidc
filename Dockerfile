# Simple approach: Use Traefik with plugin copied directly
FROM traefik:latest

# Copy plugin source to the correct local plugins directory
# Traefik expects local plugins in plugins-local/ not plugins/
COPY . /plugins-local/src/github.com/cleverunicornz/traefikoidc/

# Set working directory back to root
WORKDIR /

# Traefik will use:
# --experimental.localplugins.traefikoidc.modulename=github.com/cleverunicornz/traefikoidc
# and load from /plugins-local/src/github.com/cleverunicornz/traefikoidc

EXPOSE 80 443 8080

ENTRYPOINT ["traefik"]
