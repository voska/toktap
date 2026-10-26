package defaults

import (
	"embed"
	"os"
)

//go:embed routes.yaml pricing.yaml
var files embed.FS

// ReadFile reads an explicit path, or the named embedded default when path is empty.
func ReadFile(path, name string) ([]byte, error) {
	if path == "" {
		return files.ReadFile(name)
	}
	return os.ReadFile(path)
}
