//go:build tools

package tools

import (
	_ "go.uber.org/mock/mockgen"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/roblaszczak/go-cleanarch"
)
