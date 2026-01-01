package dream

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseResponse(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	tests := []struct {
		name     string
		input    string
		wantName string
		wantDesc bool
		wantImpl int
	}{
		{
			name: "Complete response",
			input: `Feature: Workspace Templates
Description: Allow users to create and save workspace templates with pre-configured hooks.
Rationale: Teams often work on similar types of tasks that require similar setup.
Implementation:
- Add 'ws template create <name>' to save current workspace config
- Modify 'ws new --template <name>' to create workspace from template
- Store templates in ~/.config/ws/templates/`,
			wantName: "Workspace Templates",
			wantDesc: true,
			wantImpl: 3,
		},
		{
			name: "Numbered implementation",
			input: `Feature: Auto Sync
Description: Automatically sync changes across workspaces.
Rationale: Keeps everything up to date.
Implementation:
1. Monitor file changes
2. Push to remote
3. Pull in other workspaces`,
			wantName: "Auto Sync",
			wantDesc: true,
			wantImpl: 3,
		},
		{
			name:     "Incomplete response",
			input:    "Feature: Something\nThis is not well formatted.",
			wantName: "Something",
			wantDesc: false,
			wantImpl: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parseResponse(tt.input)

			if result.Feature != tt.wantName {
				t.Errorf("Feature = %q, want %q", result.Feature, tt.wantName)
			}

			if tt.wantDesc && result.Description == "" {
				t.Error("Expected non-empty description")
			}

			if len(result.Implementation) != tt.wantImpl {
				t.Errorf("Implementation steps = %d, want %d", len(result.Implementation), tt.wantImpl)
			}
		})
	}
}

func TestExtractImplementationSteps(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name: "Bullet points with dashes",
			input: `Implementation:
- Step one
- Step two
- Step three`,
			want: 3,
		},
		{
			name: "Numbered list",
			input: `Implementation:
1. First step
2. Second step
3. Third step`,
			want: 3,
		},
		{
			name: "Mixed format",
			input: `Implementation:
- First step
* Second step
• Third step`,
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := extractImplementationSteps(tt.input)
			if len(steps) != tt.want {
				t.Errorf("extractImplementationSteps() = %d steps, want %d", len(steps), tt.want)
			}
		})
	}
}

func TestGeneratePrompt(t *testing.T) {
	cfg := &AnalyzerConfig{
		WorkingDir: "/tmp/test-repo",
	}
	analyzer := NewAnalyzer(cfg)

	prompt, err := analyzer.generatePrompt()
	if err != nil {
		t.Fatalf("generatePrompt() error = %v", err)
	}

	// Check that prompt contains key elements
	requiredElements := []string{
		"Glob",
		"Grep",
		"Feature:",
		"Description:",
		"Rationale:",
		"Implementation:",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(prompt, elem) {
			t.Errorf("Prompt missing required element: %s", elem)
		}
	}
}

func TestFormatResult(t *testing.T) {
	result := &DreamResult{
		Feature:     "Test Feature",
		Description: "A test description",
		Rationale:   "Because testing is important",
		Implementation: []string{
			"Step 1",
			"Step 2",
		},
	}

	formatted := FormatResult(result)

	if !strings.Contains(formatted, "Test Feature") {
		t.Error("Formatted result should contain feature name")
	}

	if !strings.Contains(formatted, "A test description") {
		t.Error("Formatted result should contain description")
	}

	if !strings.Contains(formatted, "Step 1") {
		t.Error("Formatted result should contain implementation steps")
	}
}

func TestAnalyzerConfig(t *testing.T) {
	cfg := DefaultAnalyzerConfig()

	if cfg.AgentCmd == "" {
		t.Error("AgentCmd should not be empty")
	}

	if cfg.Timeout == 0 {
		t.Error("Timeout should be set")
	}

	if cfg.MaxOutputBytes == 0 {
		t.Error("MaxOutputBytes should be set")
	}
}

func TestExecuteAgentTimeout(t *testing.T) {
	cfg := &AnalyzerConfig{
		AgentCmd:   "sleep 10",
		WorkingDir: "/tmp",
		Timeout:    100 * time.Millisecond,
	}
	analyzer := NewAnalyzer(cfg)

	ctx := context.Background()
	_, err := analyzer.executeAgent(ctx, "")

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") &&
	   !strings.Contains(err.Error(), "signal: killed") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}
