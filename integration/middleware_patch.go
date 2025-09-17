// This file contains the patches to be applied to Traefik's middleware system
// to register the OIDC middleware

// Add to pkg/server/middleware/middlewares.go imports:
// "github.com/traefik/traefik/v3/pkg/middlewares/oidc"

// Add to buildConstructor function in pkg/server/middleware/middlewares.go:
// "oidc": func(ctx context.Context, next http.Handler, config dynamic.Middleware, name string) (http.Handler, error) {
//     if config.Plugin == nil || config.Plugin["oidc"] == nil {
//         return nil, fmt.Errorf("oidc middleware configuration is missing")
//     }
//     cfg := config.Plugin["oidc"].(*oidc.Config)
//     return oidc.New(ctx, next, cfg, name)
// },

// Add to pkg/config/dynamic/middlewares.go in Middleware struct:
// OIDC *oidc.Config `json:"oidc,omitempty"`
