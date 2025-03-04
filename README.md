# Const Linter

A Go linter that enforces immutability of struct fields and function parameters marked with special comments.

- Detects assignments to struct fields marked with `// +const` markers 
- Detects modifications to function parameters marked as constant 
- Allows field initialization in constructor methods/functions 
- Works as a standalone command or as a golangci-lint plugin 

## Overview

"constlint" is a static analysis tool that helps you enforce immutability in your Go code by detecting unauthorized 
modifications to:

1. Struct fields marked with `// +const` comments
2. Function parameters marked with `// +const:[param1,param2,...]` directive

This linter helps prevent accidental modifications to values that should remain constant after initialization, 
improving code safety and predictability.

## Installation

### As a cli

```shell
go install github.com/bunniesandbeatings/constlint/cmd/constlint@latest
```

### With golangci-lint

TBD

# Examples

Look in the [testdata folder](./analyzer/testdata/src) for examples.