package dream_test

import (
	"context"
	"fmt"
	"log"

	"github.com/WillCMcC/ws/internal/dream"
)

// ExampleAnalyzer demonstrates basic usage of the dream analyzer
func ExampleAnalyzer() {
	// Create analyzer with default configuration
	analyzer := dream.NewAnalyzer(nil)

	// Run analysis
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Display formatted result
	fmt.Println(dream.FormatResult(result))
}

// ExampleAnalyzer_customConfig demonstrates usage with custom configuration
func ExampleAnalyzer_customConfig() {
	// Create custom configuration
	cfg := &dream.AnalyzerConfig{
		AgentCmd:   "claude --dangerously-skip-permissions",
		WorkingDir: "/path/to/project",
		Timeout:    300000000000, // 5 minutes
	}

	analyzer := dream.NewAnalyzer(cfg)

	// Run analysis with timeout
	ctx := context.Background()
	result, err := analyzer.Analyze(ctx)
	if err != nil {
		log.Printf("Analysis error: %v", err)
		// Even on error, result may contain partial data
		if result != nil {
			fmt.Println("Partial results:")
			fmt.Println(dream.FormatResult(result))
		}
		return
	}

	// Access structured data
	fmt.Printf("Feature: %s\n", result.Feature)
	fmt.Printf("Description: %s\n", result.Description)
	fmt.Printf("Implementation steps: %d\n", len(result.Implementation))
}

// ExampleAnalyzeWithDefaults shows the simplest usage pattern
func ExampleAnalyzeWithDefaults() {
	result, err := dream.AnalyzeWithDefaults("")
	if err != nil {
		log.Printf("Error: %v", err)
	}

	if result != nil {
		fmt.Println(dream.FormatResult(result))
	}
}
