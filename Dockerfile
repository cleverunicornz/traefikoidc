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

# Create the plugin integration file
RUN cat > pkg/middlewares/oidc/oidc.go << 'EOF'
package oidc

import (
	"context"
	"net/http"

	"github.com/cleverunicornz/traefikoidc"
)

// Config holds the configuration for the OIDC middleware.
type Config struct {
	ProviderURL           string            `json:"providerURL,omitempty"`
	ClientID              string            `json:"clientID,omitempty"`
	ClientSecret          string            `json:"clientSecret,omitempty"`
	CallbackURL           string            `json:"callbackURL,omitempty"`
	SessionEncryptionKey  string            `json:"sessionEncryptionKey,omitempty"`
	LogoutURL             string            `json:"logoutURL,omitempty"`
	PostLogoutRedirectURI string            `json:"postLogoutRedirectURI,omitempty"`
	Scopes                []string          `json:"scopes,omitempty"`
	AllowedUserDomains    []string          `json:"allowedUserDomains,omitempty"`
	AllowedUsers          []string          `json:"allowedUsers,omitempty"`
	AllowedRolesAndGroups []string          `json:"allowedRolesAndGroups,omitempty"`
	ForceHTTPS            *bool             `json:"forceHTTPS,omitempty"`
	LogLevel              string            `json:"logLevel,omitempty"`
	RateLimit             int               `json:"rateLimit,omitempty"`
	ExcludedURLs          []string          `json:"excludedURLs,omitempty"`
	Headers               []Header          `json:"headers,omitempty"`
	RevocationURL         string            `json:"revocationURL,omitempty"`
	OidcEndSessionURL     string            `json:"oidcEndSessionURL,omitempty"`
	EnablePKCE            *bool             `json:"enablePKCE,omitempty"`
}

// Header represents a custom header configuration
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// New creates a new OIDC middleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	// Convert our config to the plugin's config format
	pluginConfig := &traefikoidc.Config{
		ProviderURL:           config.ProviderURL,
		ClientID:              config.ClientID,
		ClientSecret:          config.ClientSecret,
		CallbackURL:           config.CallbackURL,
		SessionEncryptionKey:  config.SessionEncryptionKey,
		LogoutURL:             config.LogoutURL,
		PostLogoutRedirectURI: config.PostLogoutRedirectURI,
		Scopes:                config.Scopes,
		AllowedUserDomains:    config.AllowedUserDomains,
		AllowedUsers:          config.AllowedUsers,
		AllowedRolesAndGroups: config.AllowedRolesAndGroups,
		ForceHTTPS:            config.ForceHTTPS,
		LogLevel:              config.LogLevel,
		RateLimit:             config.RateLimit,
		ExcludedURLs:          config.ExcludedURLs,
		RevocationURL:         config.RevocationURL,
		OidcEndSessionURL:     config.OidcEndSessionURL,
		EnablePKCE:            config.EnablePKCE,
	}

	// Convert headers
	for _, h := range config.Headers {
		pluginConfig.Headers = append(pluginConfig.Headers, traefikoidc.Header{
			Name:  h.Name,
			Value: h.Value,
		})
	}

	return traefikoidc.New(ctx, next, pluginConfig, name)
}
EOF

# Register the middleware in Traefik's middleware list
RUN sed -i '/import (/a\\t"github.com/traefik/traefik/v3/pkg/middlewares/oidc"' pkg/server/middleware/middlewares.go
RUN sed -i '/func buildConstructor/a\\t\t"oidc": func(ctx context.Context, next http.Handler, config dynamic.Middleware, name string) (http.Handler, error) {\
\t\t\treturn oidc.New(ctx, next, config.Plugin["oidc"].(*oidc.Config), name)\
\t\t},' pkg/server/middleware/middlewares.go

# Add the plugin to the dynamic configuration
RUN sed -i '/Plugin map\[string\]interface{}/a\\tOIDC *oidc.Config `json:"oidc,omitempty"`' pkg/config/dynamic/middlewares.go

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
