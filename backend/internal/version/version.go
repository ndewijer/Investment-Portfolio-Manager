// Package version exposes the application version string, which is injected at compile time via ldflags.
package version //nolint:revive // var-naming: version is a standard package name

// Version is the application version string. It defaults to "dev" and is overwritten
// via ldflags at compile time (e.g., -ldflags "-X .../version.Version=1.2.3").
var Version = "dev"
