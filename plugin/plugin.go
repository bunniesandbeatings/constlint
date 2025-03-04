// Package plugin provides the plugin for golangci-lint.
package plugin

import (
	"github.com/bunniesandbeatings/constlint/analyzer"
	"golang.org/x/tools/go/analysis"
)

// AnalyzerPlugin exports the analyzer for golangci-lint.
type AnalyzerPlugin struct{}

// GetAnalyzers returns the analyzer for this plugin.
func (*AnalyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{analyzer.Analyzer}
}

// This is used by golangci-lint to identify the plugin.
var AnalyzerName = "const"
