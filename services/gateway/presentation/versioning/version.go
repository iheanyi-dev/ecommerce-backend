package versioning

// Public API versions exposed by the Gateway.
const (
	V1 = "v1"
)

// Prefix returns the public API path prefix for a version.
func Prefix(version string) string {
	return "/api/" + version
}
