package types

// DefaultVersion is the fallback version when AppContext is nil
const DefaultVersion = "dev"

// AppContext holds application-wide context information passed to commands
type AppContext struct {
	Version string
}
