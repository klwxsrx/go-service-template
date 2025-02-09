//go:build tools

package tools

import (
	_ "github.com/evilmartians/lefthook"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/jmattheis/goverter"
	_ "github.com/roblaszczak/go-cleanarch"
)
