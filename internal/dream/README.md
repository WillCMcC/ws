# Dream Analyzer

The `dream` package provides AI-powered feature analysis and suggestion capabilities for the WS task queue system.

## Overview

The Dream Analyzer uses the configured AI agent (Claude or other) to:
1. Deeply explore a codebase structure
2. Understand existing features and patterns
3. Suggest innovative, valuable features
4. Provide actionable implementation steps

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/WillCMcC/ws/internal/dream"
)

// Use default configuration
analyzer := dream.NewAnalyzer(nil)
result, err := analyzer.Analyze(context.Background())
if err != nil {
    // Handle error
}

// Display formatted result
fmt.Println(dream.FormatResult(result))
```

### Custom Configuration

```go
cfg := &dream.AnalyzerConfig{
    AgentCmd:       "claude --dangerously-skip-permissions",
    WorkingDir:     "/path/to/analyze",
    Timeout:        5 * time.Minute,
    MaxOutputBytes: 50 * 1024, // 50KB
}

analyzer := dream.NewAnalyzer(cfg)
result, err := analyzer.Analyze(context.Background())
```

### Quick Analysis

```go
// Analyze current directory with defaults
result, err := dream.AnalyzeWithDefaults("")

// Analyze specific directory
result, err := dream.AnalyzeWithDefaults("/path/to/project")
```

## Data Structures

### DreamResult

The `DreamResult` struct contains the structured output of analysis:

```go
type DreamResult struct {
    Feature        string   // Concise feature name
    Description    string   // 1-2 sentence description
    Rationale      string   // Why this feature is valuable
    Implementation []string // Specific implementation steps
    RawOutput      string   // Full agent output
    Error          string   // Error message if any (omitempty)
}
```

### AnalyzerConfig

Configuration options for the analyzer:

```go
type AnalyzerConfig struct {
    AgentCmd       string        // Command to execute (e.g., "claude")
    WorkingDir     string        // Directory to analyze
    Timeout        time.Duration // Timeout for agent execution
    MaxOutputBytes int64         // Maximum output size to capture
}
```

## How It Works

1. **Prompt Generation**: Creates a comprehensive prompt instructing the AI to:
   - Use Glob patterns to discover codebase structure
   - Use Grep to understand existing features
   - Read key files to understand architecture
   - Propose an innovative feature

2. **Agent Execution**: Runs the configured agent command with:
   - The generated prompt as stdin
   - Working directory context
   - Timeout protection
   - Output size limits

3. **Response Parsing**: Extracts structured data from AI output:
   - Feature name using regex patterns
   - Description and rationale
   - Implementation steps (supports various bullet formats)

4. **Error Handling**: Gracefully handles:
   - Command execution failures
   - Timeouts
   - Parse errors
   - Partial results on failure

## Integration with Queue GUI

The dream analyzer integrates with the WS queue GUI through:

```go
func (g *queueGUI) runDreamAnalysis() string {
    analyzer := dream.NewAnalyzer(nil)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    result, err := analyzer.Analyze(ctx)
    if err != nil {
        return fmt.Sprintf("Error: %v\n%s", err, dream.FormatResult(result))
    }

    return dream.FormatResult(result)
}
```

## Configuration

The analyzer respects the WS agent configuration:

- **Config File**: `~/.config/ws/config`
  ```
  agent_cmd=claude --dangerously-skip-permissions
  ```

- **Environment Variable**: `WS_AGENT_CMD`
  ```bash
  export WS_AGENT_CMD="claude"
  ```

- **Command**:
  ```bash
  ws config set agent_cmd "claude --dangerously-skip-permissions"
  ```

## Testing

Run tests:
```bash
go test ./internal/dream/...
```

Run with verbose output:
```bash
go test -v ./internal/dream/...
```

## Features

- ✅ Configurable AI agent command
- ✅ Timeout protection
- ✅ Comprehensive prompt generation
- ✅ Structured data extraction
- ✅ Multiple bullet format support (-, *, •, numbered)
- ✅ Graceful error handling
- ✅ Partial results on failure
- ✅ Output size limits
- ✅ Context-aware analysis
- ✅ Production-ready error handling

## Future Enhancements

Potential improvements:
- Streaming output support
- Multiple feature suggestions
- Feature voting/ranking
- Historical feature tracking
- Template-based prompts
- Language-specific analysis
