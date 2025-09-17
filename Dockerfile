# Build custom Traefik with CleverUnicornz OIDC plugin
FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Install git and make
RUN apk add --no-cache git make bash

# Clone Traefik source
RUN git clone https://github.com/traefik/traefik.git traefik-src
WORKDIR /workspace/traefik-src

# Copy our plugin source code
COPY . /workspace/plugin/

# Add our plugin as a dependency and modify go.mod
RUN go mod edit -require github.com/cleverunicornz/traefikoidc@v0.0.0
RUN go mod edit -replace github.com/cleverunicornz/traefikoidc=/workspace/plugin

# Create the OIDC middleware directory
RUN mkdir -p pkg/middlewares/oidc

# Copy our integration file
COPY integration/oidc.go pkg/middlewares/oidc/oidc.go

# Add import to middlewares.go
RUN sed -i '/^import (/a\\t"github.com/traefik/traefik/v3/pkg/middlewares/oidc"' pkg/server/middleware/middlewares.go

# Add OIDC middleware to the constructor map
RUN sed -i '/middlewareConstructors := map\[string\]Constructor{/a\\t\t"oidc": func(ctx context.Context, next http.Handler, config dynamic.Middleware, name string) (http.Handler, error) {\
    \t\t\tif config.Plugin == nil || config.Plugin["oidc"] == nil {\
    \t\t\t\treturn nil, fmt.Errorf("oidc middleware configuration is missing")\
    \t\t\t}\
    \t\t\tcfg := config.Plugin["oidc"].(*oidc.Config)\
    \t\t\treturn oidc.New(ctx, next, cfg, name)\
    \t\t},' pkg/server/middleware/middlewares.go

# Download dependencies
RUN go mod tidy

# Build Traefik
RUN make build

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /workspace/traefik-src/dist/traefik .

EXPOSE 80 443 8080

ENTRYPOINT ["./traefik"]
