//go:generate go install github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go install github.com/client9/misspell/cmd/misspell
package main

import (
	"fmt"
	"os"

	"github.com/A2va/lsw/cmd"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	// Check for ASKPASS mode immediately
	// If this env var is set, we are being called by SSH.
	// Just print the password to Stdout and exit.
	if pass, ok := os.LookupEnv("LSW_ASKPASS"); ok {
		// fmt.Print is safer than Println or Printf to avoid accidental formatting
		// of special characters in the password.
		fmt.Print(pass)
		return
	}

	cmd.Execute(
		buildVersion(version, commit, date, builtBy),
		os.Exit,
		os.Args[1:],
	)
}

func buildVersion(version, commit, date, builtBy string) string {
	var result = version
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	if builtBy != "" {
		result = fmt.Sprintf("%s\nbuilt by: %s", result, builtBy)
	}
	return result
}
