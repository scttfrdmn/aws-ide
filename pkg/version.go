package pkg

// Version represents the aws-ide platform version (shared infrastructure).
// This follows semantic versioning: MAJOR.MINOR.PATCH
//
// Breaking changes to pkg/ APIs increment MAJOR
// New features in pkg/ increment MINOR
// Bug fixes increment PATCH
//
// Apps (jupyter, rstudio, vscode) have independent versions that depend on this platform version.
const Version = "1.0.0"

// VersionInfo provides detailed version metadata
type VersionInfo struct {
	Platform string // Platform version (this package)
	Major    int    // Major version number
	Minor    int    // Minor version number
	Patch    int    // Patch version number
}

// GetVersionInfo returns structured version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Platform: Version,
		Major:    1,
		Minor:    0,
		Patch:    0,
	}
}
