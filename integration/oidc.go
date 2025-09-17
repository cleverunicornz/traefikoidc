package oidc

import (
	"context"
	"net/http"

	"github.com/cleverunicornz/traefikoidc"
)

// Config holds the configuration for the OIDC middleware.
type Config struct {
	ProviderURL           string   `json:"providerURL,omitempty"`
	ClientID              string   `json:"clientID,omitempty"`
	ClientSecret          string   `json:"clientSecret,omitempty"`
	CallbackURL           string   `json:"callbackURL,omitempty"`
	SessionEncryptionKey  string   `json:"sessionEncryptionKey,omitempty"`
	LogoutURL             string   `json:"logoutURL,omitempty"`
	PostLogoutRedirectURI string   `json:"postLogoutRedirectURI,omitempty"`
	Scopes                []string `json:"scopes,omitempty"`
	AllowedUserDomains    []string `json:"allowedUserDomains,omitempty"`
	AllowedUsers          []string `json:"allowedUsers,omitempty"`
	AllowedRolesAndGroups []string `json:"allowedRolesAndGroups,omitempty"`
	ForceHTTPS            *bool    `json:"forceHTTPS,omitempty"`
	LogLevel              string   `json:"logLevel,omitempty"`
	RateLimit             int      `json:"rateLimit,omitempty"`
	ExcludedURLs          []string `json:"excludedURLs,omitempty"`
	Headers               []Header `json:"headers,omitempty"`
	RevocationURL         string   `json:"revocationURL,omitempty"`
	OidcEndSessionURL     string   `json:"oidcEndSessionURL,omitempty"`
	EnablePKCE            *bool    `json:"enablePKCE,omitempty"`
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
