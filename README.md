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

To use constlint with golangci-lint, you need to configure it as a plugin. Follow these steps:

1. First, install golangci-lint if you haven't already:
   ```shell
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

2. Build the constlint plugin:
   ```shell
   go build -buildmode=plugin -o plugin.so ./plugin
   ```

3. Configure golangci-lint to use the plugin by adding the following to your `.golangci.yml` file:
   ```yaml
   linters-settings:
     custom:
       constlint:
         path: github.com/bunniesandbeatings/constlint/plugin.so
         description: Checks for writes to struct fields marked with // +const
         original-url: github.com/bunniesandbeatings/constlint

   linters:
     enable:
       - constlint
   ```

4. Run golangci-lint as usual:
   ```shell
   golangci-lint run
   ```

Note: The plugin must be built for the same architecture and Go version as golangci-lint. If you encounter any issues, try rebuilding the plugin with the same Go version used by golangci-lint.

# Examples

Look in the [testdata folder](./analyzer/testdata/src) for examples.
