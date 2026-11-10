// Package migrations embeds the SQL migration files into the binary so that
// the application can migrate itself without shipping the .sql files.
//
// Create a new migration with:
//
//	go run . migrate create <name>
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
