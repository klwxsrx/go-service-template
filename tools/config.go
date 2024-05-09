//go:build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/roblaszczak/go-cleanarch"
	_ "github.com/jmattheis/goverter"
	_ "go.uber.org/mock/mockgen"
)
