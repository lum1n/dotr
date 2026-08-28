// Package version holds build-time identity for dotr.
package version

import "fmt"

// Set via -ldflags at release build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable version line.
func String() string {
	if Commit == "none" || Commit == "" {
		return Version
	}
	short := Commit
	if len(short) > 7 {
		short = short[:7]
	}
	return fmt.Sprintf("%s (%s)", Version, short)
}

// Full returns version, commit, and build date.
func Full() string {
	return fmt.Sprintf("dotr %s\ncommit: %s\nbuilt:  %s", Version, Commit, Date)
}
