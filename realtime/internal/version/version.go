// Package version is the single source of truth for the realtime engine build
// version. It is stamped via -ldflags "-X ...version.Version=x.y.z" at build
// time (Makefile go-build) so /health, telemetry and log lines report the real
// release instead of the stale hardcoded "1.0.0" that masked drift for weeks.
package version

// Version is the current engine release. Update on each release tag together
// with the README version line.
var Version = "1.18.0"