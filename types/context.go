package types

// DefaultVersion is the fallback version when AppContext is nil
const DefaultVersion = "dev"

// AppContext holds application-wide context information passed to commands
type AppContext struct {
	// Version is the application version string (e.g., "1.0.0" or "dev")
	Version string
}
