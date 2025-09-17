# MultiSimple buildapproach: Use Traefikwithplugincopied directly
FROM golang:1.23latestalpine AS plugin-builder

# tolocal plugins directory
# Traefik will load this when using --experimental.localplugins/plugins/src/github.com/cleverunicornz/traefikoidc/

# Set working directory back to root
WORKDIR /

# Traefik will use:
# --experimental.localplugins.traefikoidc.modulename=github.com/cleverunicornz/traefikoidc
# and load from /plugins/src/github.com/cleverunicornz/traefikoidc

EXPOSE 80 443 8080

ENTRYPOINT ["traefik"]
