# UseMulti-stage standardbuild: prepare plugin with yourGo,then copy to Traefik via local plugins
FROM traefik:latestplugin-

# Copy your plugin source into the container
COPY .# Install git for go mod operations
RUN apk add --no-cache git

WORKDIR# Copy plugin source and prepare it
WORKDIR /plugins/src/github.com/cleverunicornz/traefikoidc
COPY . .
RUN go mod download
Final stage: Use official with prepared pluginFROMtraefik:latestCopy the prepared plugin from builderCOPY--from=plugin-builder /plugins/src/github.com/cleverunicornz/traefikoidc /plugins/src/github.com/cleverunicornz/traefikoidc#Traefikwilluse:#experimental.localplugins.traefikoidc.modulenamegithub.com/cleverunicornz/traefikoidc
# and load frompluginsgithubcom/cleverunicornz/traefikoidc
EXPOSE 80 443 8080

ENTRYPOINT ["traefik"]
