// Ayaka Backend
//
//	@title						Ayaka Backend API
//	@version					1.0
//	@description				Production-ready Go backend template (Fiber + GORM + PostgreSQL).
//	@BasePath					/
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and the access token.
package main

import (
	"fmt"
	"os"

	"be-ayaka/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
