package cmd

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/crymfox/nac/internal/config"
	"github.com/spf13/cobra"
)

//go:embed templates/*
var templateFS embed.FS

// initData holds template variables for scaffolding.
type initData struct {
	N8NVersion string
}

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new nac project in the current directory",
		Long: `Creates the nac project structure in the current directory:
  - nac.yaml              (main config file)
  - docker-compose.yaml   (local n8n + Postgres + Redis stack)
  - .env.local.example    (local environment template)
  - .env.remote.example   (remote environment template)
  - .gitignore            (ignore patterns for n8n files)
  - n8n_workflows/        (workflow JSON directory)
  - n8n_credentials/      (credential JSON directory)

Optionally generates a GitHub Actions CI workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")
	return cmd
}

func runInit(force bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	data := initData{
		N8NVersion: config.PinnedN8NVersion,
	}

	// Files to generate from templates
	files := []struct {
		tmpl string
		dest string
		desc string
	}{
		{"templates/nac.yaml.tmpl", "nac.yaml", "config file"},
		{"templates/docker-compose.yaml.tmpl", "docker-compose.yaml", "Docker Compose stack"},
		{"templates/env.local.example.tmpl", ".env.local.example", "local environment template"},
		{"templates/env.remote.example.tmpl", ".env.remote.example", "remote environment template"},
	}

	// Check for existing files first (unless --force)
	if !force {
		for _, f := range files {
			dest := filepath.Join(dir, f.dest)
			if _, err := os.Stat(dest); err == nil {
				return fmt.Errorf("file already exists: %s (use --force to overwrite)", f.dest)
			}
		}
	}

	// Create directories
	dirs := []string{"n8n_workflows", "n8n_credentials"}
	for _, d := range dirs {
		path := filepath.Join(dir, d)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
		// Add .gitkeep so empty dirs are tracked
		gitkeep := filepath.Join(path, ".gitkeep")
		if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
			if err := os.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
				return fmt.Errorf("creating %s: %w", gitkeep, err)
			}
		}
		fmt.Printf("  created %s/\n", d)
	}

	// Render templates
	for _, f := range files {
		if err := renderTemplate(f.tmpl, filepath.Join(dir, f.dest), data); err != nil {
			return fmt.Errorf("generating %s: %w", f.dest, err)
		}
		fmt.Printf("  created %s (%s)\n", f.dest, f.desc)
	}

	// Handle .gitignore: append rather than overwrite
	if err := appendGitignore(dir); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}
	fmt.Println("  updated .gitignore")

	// Ask about GitHub Actions
	if askYesNo("Generate GitHub Actions CI workflow?", true) {
		ghDir := filepath.Join(dir, ".github", "workflows")
		if err := os.MkdirAll(ghDir, 0o755); err != nil {
			return fmt.Errorf("creating .github/workflows: %w", err)
		}
		dest := filepath.Join(ghDir, "deploy-n8n.yml")
		if err := renderTemplate("templates/github-actions.yaml.tmpl", dest, data); err != nil {
			return fmt.Errorf("generating GitHub Actions workflow: %w", err)
		}
		fmt.Println("  created .github/workflows/deploy-n8n.yml")
	}

	fmt.Println()
	fmt.Println("nac project initialized. Next steps:")
	fmt.Println()
	fmt.Println("  1. Copy .env.local.example to .env.local and fill in your values")
	fmt.Println("  2. Start the local stack:  nac up")
	fmt.Println("  3. Build workflows in n8n at http://localhost:5678")
	fmt.Println("  4. Export to files:         nac export workflows")
	fmt.Println("  5. Commit and push to deploy via CI")

	return nil
}

func renderTemplate(tmplPath, destPath string, data initData) error {
	content, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("reading embedded template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", tmplPath, err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", destPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template %s: %w", tmplPath, err)
	}

	return nil
}

func appendGitignore(dir string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	// Read the template content
	tmplContent, err := templateFS.ReadFile("templates/gitignore.tmpl")
	if err != nil {
		return fmt.Errorf("reading gitignore template: %w", err)
	}

	nacBlock := fmt.Sprintf("\n# === nac (n8n As Code) ===\n%s\n", string(tmplContent))

	// Check if .gitignore exists and already has nac section
	if existing, err := os.ReadFile(gitignorePath); err == nil {
		if strings.Contains(string(existing), "# === nac (n8n As Code) ===") {
			// Already has our section, skip
			return nil
		}
		// Append to existing
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(nacBlock)
		return err
	}

	// Create new .gitignore
	return os.WriteFile(gitignorePath, []byte(nacBlock), 0o644)
}

func askYesNo(question string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	suffix := " [Y/n]: "
	if !defaultYes {
		suffix = " [y/N]: "
	}

	fmt.Print(question + suffix)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}
