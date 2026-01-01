package dream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/WillCMcC/ws/internal/config"
)

// DreamResult represents the structured output of the dream analysis
type DreamResult struct {
	Feature        string   `json:"feature"`
	Description    string   `json:"description"`
	Rationale      string   `json:"rationale"`
	Implementation []string `json:"implementation"`
	RawOutput      string   `json:"raw_output"`
	Error          string   `json:"error,omitempty"`
}

// AnalyzerConfig holds configuration for the dream analyzer
type AnalyzerConfig struct {
	AgentCmd       string        // Command to execute (e.g., "claude")
	WorkingDir     string        // Directory to analyze
	Timeout        time.Duration // Timeout for agent execution
	MaxOutputBytes int64         // Maximum output size to capture
}

// DefaultAnalyzerConfig returns the default analyzer configuration
func DefaultAnalyzerConfig() *AnalyzerConfig {
	cfg := config.Load()
	wd, _ := os.Getwd()

	return &AnalyzerConfig{
		AgentCmd:       cfg.Agent.Cmd,
		WorkingDir:     wd,
		Timeout:        5 * time.Minute,
		MaxOutputBytes: 50 * 1024, // 50KB
	}
}

// Analyzer performs dream analysis on a codebase
type Analyzer struct {
	config *AnalyzerConfig
}

// NewAnalyzer creates a new dream analyzer with the given configuration
func NewAnalyzer(cfg *AnalyzerConfig) *Analyzer {
	if cfg == nil {
		cfg = DefaultAnalyzerConfig()
	}
	return &Analyzer{config: cfg}
}

// Analyze performs a deep analysis of the codebase and suggests an innovative feature
func (a *Analyzer) Analyze(ctx context.Context) (*DreamResult, error) {
	// Generate the analysis prompt
	prompt, err := a.generatePrompt()
	if err != nil {
		return &DreamResult{Error: fmt.Sprintf("failed to generate prompt: %v", err)}, err
	}

	// Execute the agent with the prompt
	output, err := a.executeAgent(ctx, prompt)
	if err != nil {
		return &DreamResult{
			RawOutput: string(output),
			Error:     fmt.Sprintf("agent execution failed: %v", err),
		}, err
	}

	// Parse the agent's response
	result := a.parseResponse(string(output))
	result.RawOutput = string(output)

	return result, nil
}

// generatePrompt creates a comprehensive prompt for the AI agent
func (a *Analyzer) generatePrompt() (string, error) {
	// Get repository information
	repoName := filepath.Base(a.config.WorkingDir)

	// Check if git repository
	isGitRepo := false
	if _, err := os.Stat(filepath.Join(a.config.WorkingDir, ".git")); err == nil {
		isGitRepo = true
	}

	prompt := fmt.Sprintf(`You are a creative software architect analyzing the "%s" codebase to suggest innovative features.

TASK: Deeply explore this codebase and propose ONE specific, valuable feature that would significantly enhance the tool.

ANALYSIS APPROACH:
1. Use Glob patterns to discover the codebase structure:
   - Search for "**/*.go" to find all Go source files
   - Search for "**/cmd/**/*.go" to find command implementations
   - Search for "**/internal/**/*.go" to find internal packages
   - Search for "*.md" to find documentation

2. Use Grep to understand existing features and patterns:
   - Search for "type.*struct" to understand data models
   - Search for "func.*Cmd" to find available commands
   - Search for "TODO|FIXME|XXX" to find known gaps
   - Search for "interface" to understand abstractions

3. Read key files to understand:
   - Architecture and design patterns
   - Current features and workflows
   - Configuration options
   - User-facing commands

4. Consider:
   - What pain points might users experience?
   - What workflows could be streamlined?
   - What features would make this tool indispensable?
   - What innovative capabilities are missing?

CONSTRAINTS:
- The feature should be practical and implementable
- It should align with the tool's existing architecture
- It should provide clear value to users
- Be creative but realistic

OUTPUT FORMAT (REQUIRED):
Feature: [Concise feature name, 2-5 words]
Description: [Clear 1-2 sentence description of what the feature does]
Rationale: [2-3 sentences explaining why this is valuable and what problem it solves]
Implementation:
- [Step 1: Specific implementation step]
- [Step 2: Another implementation step]
- [Step 3: Continue with detailed steps]
- [Step N: As many steps as needed]

IMPORTANT:
- You MUST use the Glob and Grep tools to explore the codebase first
- You MUST read at least 3-5 key files to understand the architecture
- Be specific and actionable in your implementation steps
- Think beyond obvious features - be innovative!
`, repoName)

	if isGitRepo {
		prompt += "\nNOTE: This is a git repository. Consider git-integrated features.\n"
	}

	prompt += "\nBegin your analysis now. Remember to explore thoroughly before suggesting!"

	return prompt, nil
}

// executeAgent runs the configured agent command with the given prompt
func (a *Analyzer) executeAgent(ctx context.Context, prompt string) ([]byte, error) {
	// Create a context with timeout
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	// Parse agent command (handle commands with arguments)
	cmdParts := strings.Fields(a.config.AgentCmd)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("agent command is empty")
	}

	cmdName := cmdParts[0]
	cmdArgs := cmdParts[1:]

	// Create the command
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = a.config.WorkingDir

	// Set up stdin with the prompt
	cmd.Stdin = strings.NewReader(prompt)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_BETA=max-tokens-3-5-sonnet-2024-07-15",
	)

	// Execute the command
	err := cmd.Run()

	// Combine stdout and stderr for output
	output := stdout.Bytes()

	if err != nil {
		// Include stderr in error for debugging
		if stderr.Len() > 0 {
			return output, fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
		}
		return output, fmt.Errorf("command failed: %w", err)
	}

	// Check output size
	if a.config.MaxOutputBytes > 0 && int64(len(output)) > a.config.MaxOutputBytes {
		output = output[:a.config.MaxOutputBytes]
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("agent produced no output")
	}

	return output, nil
}

// parseResponse extracts structured data from the agent's response
func (a *Analyzer) parseResponse(output string) *DreamResult {
	result := &DreamResult{}

	// Clean up the output
	output = strings.TrimSpace(output)

	// Extract Feature
	if matches := extractField(output, `Feature:\s*(.+?)(?:\n|$)`); len(matches) > 1 {
		result.Feature = strings.TrimSpace(matches[1])
	}

	// Extract Description
	if matches := extractField(output, `Description:\s*(.+?)(?:\n(?:Rationale|Implementation):|$)`); len(matches) > 1 {
		result.Description = strings.TrimSpace(matches[1])
	}

	// Extract Rationale
	if matches := extractField(output, `Rationale:\s*(.+?)(?:\n(?:Implementation):|$)`); len(matches) > 1 {
		result.Rationale = strings.TrimSpace(matches[1])
	}

	// Extract Implementation steps
	result.Implementation = extractImplementationSteps(output)

	// Validation
	if result.Feature == "" {
		result.Feature = "Unnamed Feature"
		if result.Error == "" {
			result.Error = "Could not parse feature name from response"
		}
	}

	if result.Description == "" {
		result.Description = "No description available"
		if result.Error == "" {
			result.Error = "Could not parse description from response"
		}
	}

	return result
}

// extractField extracts a field value using a regex pattern
func extractField(text, pattern string) []string {
	re := regexp.MustCompile(`(?s)` + pattern)
	matches := re.FindStringSubmatch(text)
	return matches
}

// extractImplementationSteps parses implementation steps from the output
func extractImplementationSteps(output string) []string {
	var steps []string

	// Find the Implementation section
	re := regexp.MustCompile(`(?s)Implementation:\s*\n(.*?)(?:\n\n|$)`)
	matches := re.FindStringSubmatch(output)

	if len(matches) < 2 {
		return steps
	}

	implText := matches[1]

	// Extract bullet points (-, *, or numbered)
	lines := strings.Split(implText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Match various bullet point formats
		if matched, _ := regexp.MatchString(`^[-*•]\s+.+`, line); matched {
			// Remove bullet point
			step := regexp.MustCompile(`^[-*•]\s+`).ReplaceAllString(line, "")
			steps = append(steps, strings.TrimSpace(step))
		} else if matched, _ := regexp.MatchString(`^\d+[\.)]\s+.+`, line); matched {
			// Remove numbered list marker
			step := regexp.MustCompile(`^\d+[\.)]\s+`).ReplaceAllString(line, "")
			steps = append(steps, strings.TrimSpace(step))
		} else if line != "" && !strings.Contains(line, ":") {
			// Plain text that's not a header
			steps = append(steps, line)
		}
	}

	return steps
}

// AnalyzeWithDefaults performs analysis using default configuration
func AnalyzeWithDefaults(workingDir string) (*DreamResult, error) {
	cfg := DefaultAnalyzerConfig()
	if workingDir != "" {
		cfg.WorkingDir = workingDir
	}

	analyzer := NewAnalyzer(cfg)
	return analyzer.Analyze(context.Background())
}

// FormatResult formats a DreamResult as a human-readable string
func FormatResult(result *DreamResult) string {
	var buf strings.Builder

	buf.WriteString("🌟 Dream Analysis Complete!\n\n")

	if result.Error != "" {
		buf.WriteString(fmt.Sprintf("⚠️  Warning: %s\n\n", result.Error))
	}

	buf.WriteString(fmt.Sprintf("Feature: %s\n", result.Feature))
	buf.WriteString(fmt.Sprintf("Description: %s\n", result.Description))

	if result.Rationale != "" {
		buf.WriteString(fmt.Sprintf("Rationale: %s\n", result.Rationale))
	}

	if len(result.Implementation) > 0 {
		buf.WriteString("\nImplementation:\n")
		for _, step := range result.Implementation {
			buf.WriteString(fmt.Sprintf("- %s\n", step))
		}
	}

	return buf.String()
}
