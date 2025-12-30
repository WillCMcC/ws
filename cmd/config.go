package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type configItem struct {
	key         string
	description string
	example     string
}

var configItems = []configItem{
	{
		key:         "agent_cmd",
		description: "Command to run when using 'ws ez'. This starts your AI coding agent in the new workspace.",
		example:     "claude --dangerously-skip-permissions",
	},
	{
		key:         "default_base",
		description: "Default branch to use as base when creating new workspaces. Leave empty to auto-detect (main/master).",
		example:     "develop",
	},
	{
		key:         "directory",
		description: "Pattern for workspace directory location. Use {repo} as placeholder for repository name.",
		example:     "../{repo}-ws",
	},
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	tooltipStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).
			MarginTop(1)

	exampleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

type model struct {
	items    []configItem
	cursor   int
	config   map[string]string
	editing  bool
	input    string
	quitting bool
	saved    bool
	message  string
}

func initialModel() model {
	config, _ := loadConfigFile()
	if config == nil {
		config = make(map[string]string)
	}
	return model{
		items:  configItems,
		config: config,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			return m.handleEditingInput(msg)
		}
		return m.handleNavigationInput(msg)
	}
	return m, nil
}

func (m model) handleNavigationInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "enter":
		m.editing = true
		// Pre-fill with current value
		m.input = m.config[m.items[m.cursor].key]
		m.message = ""
	}
	return m, nil
}

func (m model) handleEditingInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editing = false
		m.input = ""
		m.message = ""
	case "enter":
		// Save the value
		key := m.items[m.cursor].key
		m.config[key] = m.input
		if err := saveConfigFile(m.config); err != nil {
			m.message = fmt.Sprintf("Error: %v", err)
		} else {
			if m.input == "" {
				m.message = fmt.Sprintf("Cleared %s", key)
			} else {
				m.message = fmt.Sprintf("Saved %s", key)
			}
			m.saved = true
		}
		m.editing = false
		m.input = ""
	case "backspace", "ctrl+h":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	case "ctrl+u":
		m.input = ""
	default:
		// Only add printable characters
		if len(msg.String()) == 1 || msg.String() == " " {
			m.input += msg.String()
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("ws config"))
	b.WriteString("\n\n")

	// List items
	for i, item := range m.items {
		cursor := "  "
		style := normalStyle

		if i == m.cursor {
			cursor = "▸ "
			style = selectedStyle
		}

		value := m.config[item.key]
		if value == "" {
			value = "(not set)"
		}

		line := fmt.Sprintf("%s%s", cursor, style.Render(item.key))
		b.WriteString(line)
		b.WriteString("  ")
		b.WriteString(valueStyle.Render(value))
		b.WriteString("\n")
	}

	// Tooltip for selected item
	selected := m.items[m.cursor]
	tooltip := selected.description + "\n" + exampleStyle.Render("Example: "+selected.example)
	b.WriteString(tooltipStyle.Render(tooltip))
	b.WriteString("\n")

	// Input area or message
	if m.editing {
		b.WriteString("\n")
		b.WriteString(promptStyle.Render("Enter value: "))
		b.WriteString(inputStyle.Render(m.input))
		b.WriteString("█")
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter: save • esc: cancel • ctrl+u: clear"))
	} else if m.message != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓ " + m.message))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: edit • q: quit"))
	} else {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: edit • q: quit"))
	}

	return b.String()
}

// ConfigCmd handles the 'ws config' command.
func ConfigCmd(args []string) int {
	fs := flag.NewFlagSet("config", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws config [subcommand] [arguments]\n\n")
		fmt.Fprintf(os.Stderr, "Manage ws configuration.\n\n")
		fmt.Fprintf(os.Stderr, "Without arguments, opens interactive config editor.\n\n")
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  set <key> <value>   Set a configuration value\n")
		fmt.Fprintf(os.Stderr, "  get <key>           Get a configuration value\n")
		fmt.Fprintf(os.Stderr, "  list                List all configuration values\n")
		fmt.Fprintf(os.Stderr, "  path                Show config file path\n\n")
		fmt.Fprintf(os.Stderr, "Available keys:\n")
		for _, item := range configItems {
			fmt.Fprintf(os.Stderr, "  %-14s %s\n", item.key, item.description)
		}
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  ws config                                              # Interactive mode\n")
		fmt.Fprintf(os.Stderr, "  ws config set agent_cmd \"claude --dangerously-skip-permissions\"\n")
		fmt.Fprintf(os.Stderr, "  ws config get agent_cmd\n")
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// No arguments - run interactive mode
	if fs.NArg() < 1 {
		return runInteractiveConfig()
	}

	subCmd := fs.Arg(0)
	subArgs := fs.Args()[1:]

	switch subCmd {
	case "set":
		return configSet(subArgs)
	case "get":
		return configGet(subArgs)
	case "list":
		return configList()
	case "path":
		fmt.Println(getConfigPath())
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ws config: unknown subcommand '%s'\n", subCmd)
		fs.Usage()
		return 1
	}
}

func runInteractiveConfig() int {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func configSet(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: ws config set <key> <value>\n")
		return 1
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	if !isValidKey(key) {
		fmt.Fprintf(os.Stderr, "ws config: unknown key '%s'\n", key)
		fmt.Fprintf(os.Stderr, "Available keys: %s\n", strings.Join(getConfigKeyNames(), ", "))
		return 1
	}

	config, err := loadConfigFile()
	if err != nil {
		config = make(map[string]string)
	}

	config[key] = value

	if err := saveConfigFile(config); err != nil {
		fmt.Fprintf(os.Stderr, "ws config: failed to save config: %v\n", err)
		return 1
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return 0
}

func configGet(args []string) int {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: ws config get <key>\n")
		return 1
	}

	key := args[0]

	if !isValidKey(key) {
		fmt.Fprintf(os.Stderr, "ws config: unknown key '%s'\n", key)
		fmt.Fprintf(os.Stderr, "Available keys: %s\n", strings.Join(getConfigKeyNames(), ", "))
		return 1
	}

	config, err := loadConfigFile()
	if err != nil {
		fmt.Println("(not set)")
		return 0
	}

	if value, ok := config[key]; ok {
		fmt.Println(value)
	} else {
		fmt.Println("(not set)")
	}
	return 0
}

func configList() int {
	config, _ := loadConfigFile()

	fmt.Println("ws configuration:")
	fmt.Println()

	for _, item := range configItems {
		value := "(not set)"
		if v, ok := config[item.key]; ok && v != "" {
			value = v
		}
		fmt.Printf("  %s\n", item.key)
		fmt.Printf("    %s\n", item.description)
		fmt.Printf("    Value: %s\n\n", value)
	}

	fmt.Printf("Config file: %s\n", getConfigPath())
	return 0
}

func isValidKey(key string) bool {
	for _, item := range configItems {
		if item.key == key {
			return true
		}
	}
	return false
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ws", "config")
}

func loadConfigFile() (map[string]string, error) {
	path := getConfigPath()
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return config, scanner.Err()
}

func saveConfigFile(config map[string]string) error {
	path := getConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for key, value := range config {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return nil
}

func getConfigKeyNames() []string {
	keys := make([]string, 0, len(configItems))
	for _, item := range configItems {
		keys = append(keys, item.key)
	}
	return keys
}

// GetConfigValue returns a config value, checking file config then env vars.
func GetConfigValue(key string) string {
	// First check config file
	config, err := loadConfigFile()
	if err == nil {
		if value, ok := config[key]; ok {
			return value
		}
	}

	// Fall back to environment variables
	switch key {
	case "agent_cmd":
		return os.Getenv("WS_AGENT_CMD")
	case "default_base":
		return os.Getenv("WS_DEFAULT_BASE")
	case "directory":
		return os.Getenv("WS_DIRECTORY")
	}

	return ""
}
