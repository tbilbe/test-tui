# Chapter 1: Project Setup & Go Modules

## What You'll Learn

- How to initialise a Go project
- What `go.mod` does and why it matters
- The standard project layout
- Why we structure directories the way we do

---

## Starting From Nothing

Open your terminal and create a new directory:

```bash
mkdir seven-test-tui
cd seven-test-tui
```

Now initialise a Go module:

```bash
go mod init github.com/angstromsports/seven-test-tui
```

This creates a `go.mod` file:

```go
module github.com/angstromsports/seven-test-tui

go 1.25.0
```

### What is `go.mod`?

Think of `go.mod` as your project's identity card. It tells Go:

1. **Module path**: The unique name for your project (`github.com/angstromsports/seven-test-tui`)
2. **Go version**: Which Go version this project uses
3. **Dependencies**: What external packages you need (added automatically when you import them)

The module path is typically your repository URL. This matters because:
- Other projects can import your code using this path
- Go knows where to fetch your code from

---

## The Standard Project Layout

Create this directory structure:

```bash
mkdir -p cmd
mkdir -p internal/aws
mkdir -p internal/models
mkdir -p internal/ui
mkdir -p pkg/config
```

Your project now looks like:

```
seven-test-tui/
├── cmd/              # Application entry points
├── internal/         # Private application code
│   ├── aws/          # AWS service clients
│   ├── models/       # Domain types
│   └── ui/           # TUI components
├── pkg/              # Public library code
│   └── config/       # Configuration management
└── go.mod
```

### Why This Structure?

**`cmd/`** - Contains your `main.go` files. If you had multiple executables (like a CLI tool and a server), each would get its own subdirectory here.

**`internal/`** - This is special in Go. Code inside `internal/` cannot be imported by external projects. The Go compiler enforces this. Use it for code that's specific to your application.

**`pkg/`** - Code here can be imported by other projects. Use it for reusable utilities. Some teams skip this and put everything in `internal/` - that's fine too.

### The `internal/` Guarantee

This is important. If you write:

```go
// In some external project
import "github.com/angstromsports/seven-test-tui/internal/models"
```

The Go compiler will refuse to build. This protects your implementation details.

---

## Adding Dependencies

We need the Bubbletea TUI framework and AWS SDK. Add them:

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
```

Your `go.mod` now has a `require` block listing these dependencies. Go also creates `go.sum` - a lockfile ensuring reproducible builds.

---

## Creating the Entry Point

Create `cmd/main.go`:

```go
package main

import "fmt"

func main() {
    fmt.Println("Seven Test TUI")
}
```

Run it:

```bash
go run cmd/main.go
```

You should see:

```
Seven Test TUI
```

Congratulations. You have a working Go application.

---

## Key Takeaways

1. **`go mod init`** creates your project identity
2. **`internal/`** protects your code from external imports
3. **`cmd/`** holds entry points, keep them thin
4. **`go get`** adds dependencies to `go.mod`
5. **`go run`** compiles and runs in one step

---

## Exercise

1. Try importing something from `internal/` in a separate Go project. Watch it fail.
2. Run `go mod tidy` - this removes unused dependencies and adds missing ones.
3. Look at `go.sum` - notice how it pins exact versions.

---

[← Previous: Introduction](./00-introduction.md) | [Next: Chapter 2 - Understanding Structs →](./02-structs.md)
