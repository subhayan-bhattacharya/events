// Package migrations exposes the production goose migrations to Go code.
package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var files embed.FS

// Files returns the production SQL migrations rooted at this package.
func Files() fs.FS {
	return files
}
