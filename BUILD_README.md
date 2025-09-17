# Custom Traefik with CleverUnicornz OIDC Plugin

This repository contains a custom build of Traefik that includes the CleverUnicornz OIDC plugin as a built-in middleware, bypassing the need for the official Traefik plugin registry.

## What This Does

- Clones the official Traefik source code
- Integrates the CleverUnicornz OIDC plugin as a native middleware
- Builds a custom Traefik binary with the plugin embedded
- Publishes the custom image to GitHub Container Registry

## Docker Image

The custom Traefik image is automatically built and published to:
```
ghcr.io/cleverunicornz/traefik-oidc:latest
```

Available tags:
- `latest` - Latest build from fix/math-rand-import branch
- `fix-math-rand-import` - Branch-specific builds
- `sha-<commit>` - Commit-specific builds
- `v*` - Version tags when tagged

## Usage in Nomad

```hcl
config {
  image = "ghcr.io/cleverunicornz/traefik-oidc:latest"
  # No experimental plugin configuration needed!
}
```

## Middleware Configuration

The OIDC middleware is available as a built-in middleware named `oidc`:

```yaml
http:
  middlewares:
    oidc:
      oidc:
        providerURL: "http://172.16.1.1:8280/v1/identity/oidc"
        clientID: "your-client-id"
        clientSecret: "your-client-secret"
        callbackURL: "http://172.16.1.1:8085/callback"
        sessionEncryptionKey: "32-byte-encryption-key"
        # ... other OIDC configuration
```

## Local Testing

To test the build locally:

```bash
docker build -t custom-traefik-oidc .
docker run -p 8085:8085 custom-traefik-oidc --help
```

## Build Process

The GitHub Actions workflow:
1. Triggers on push to `fix/math-rand-import` branch
2. Clones official Traefik repository
3. Copies this plugin source code into the build
4. Modifies Traefik to register the OIDC plugin as built-in middleware
5. Builds custom Traefik binary
6. Creates Docker image and pushes to GHCR

## Why This Approach

Traefik's experimental plugin system only works with plugins in their official registry. Since the CleverUnicornz fork contains fixes not in the original plugin, we bypass the plugin system entirely by embedding the plugin directly into Traefik at build time.

This gives us:
- ✅ Full control over the plugin version
- ✅ No dependency on Traefik's plugin registry
- ✅ Fixes included from the fix/math-rand-import branch
- ✅ Native performance (no plugin overhead)
- ✅ Automated builds via GitHub Actions