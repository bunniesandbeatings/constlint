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

### With golangci-lint (Module Plugin)

To use constlint with golangci-lint as a module plugin, follow these steps:

#### The Automatic Way (Recommended)

1. Create a `.custom-gcl.yml` file in your project with the following content:
   ```yaml
   version: v1.64.6  # Use your preferred golangci-lint version
   plugins:
     - module: 'github.com/bunniesandbeatings/constlint'
       import: 'github.com/bunniesandbeatings/constlint/plugin'
       version: v1.0.0  # Use the appropriate version
   ```

2. Configure golangci-lint to use the plugin by adding the following to your `.golangci.yml` file:
   ```yaml
   linters-settings:
     custom:
       constlint:
         type: "module"
         description: Checks for writes to struct fields marked with // +const

   linters:
     enable:
       - constlint
   ```

3. Build your custom golangci-lint binary:
   ```shell
   golangci-lint custom
   ```

4. Run your custom golangci-lint:
   ```shell
   ./custom-gcl run
   ```

#### The Manual Way

1. Clone the golangci-lint repository:
   ```shell
   git clone https://github.com/golangci/golangci-lint.git
   cd golangci-lint
   ```

2. Add a blank import for the constlint plugin in `cmd/golangci-lint/plugins.go`:
   ```go
   import (
       // Add other imports...
       
       // Add constlint plugin
       _ "github.com/bunniesandbeatings/constlint/plugin"
   )
   ```

3. Run `go mod tidy` and build golangci-lint:
   ```shell
   go mod tidy
   make build
   ```

4. Configure your `.golangci.yml` file as shown in the automatic method above.

5. Run your custom golangci-lint binary.

# Examples

Look in the [testdata folder](./analyzer/testdata/src) for examples.
