package cmd

import (
	"bufio"
	"crypto/rand"
	"embed"
	"encoding/hex"
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
		Long:  `Creates the nac project structure...`, // Shortened for brevity
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

	filesToRender := []struct {
		tmpl string
		dest string
		desc string
	}{
		{"templates/nac.yaml.tmpl", "nac.yaml", "config file"},
		{"templates/docker-compose.yaml.tmpl", "docker-compose.yaml", "Docker Compose stack"},
		{"templates/env.remote.example.tmpl", ".env.remote.example", "remote environment template"},
	}

	if !force {
		// Check for all files at once before writing anything
		for _, f := range filesToRender {
			if _, err := os.Stat(filepath.Join(dir, f.dest)); err == nil {
				return fmt.Errorf("file already exists: %s (use --force to overwrite)", f.dest)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, ".env.local")); err == nil {
			return fmt.Errorf("file already exists: .env.local (use --force to overwrite)")
		}
	}

	// Create directories
	dirs := []string{"n8n_workflows", "n8n_credentials"}
	for _, d := range dirs {
		path := filepath.Join(dir, d)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
		gitkeep := filepath.Join(path, ".gitkeep")
		if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
			_ = os.WriteFile(gitkeep, []byte{}, 0o644)
		}
		fmt.Printf("  created %s/\n", d)
	}

	// Render standard templates
	for _, f := range filesToRender {
		if err := renderTemplate(f.tmpl, filepath.Join(dir, f.dest), data); err != nil {
			return fmt.Errorf("generating %s: %w", f.dest, err)
		}
		fmt.Printf("  created %s (%s)\n", f.dest, f.desc)
	}

	// Special handling for .env.local (generate key)
	if err := createDotEnvLocal(dir, force); err != nil {
		return err
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

	fmt.Println("\nnac project initialized. Next steps:")
	fmt.Println("\n  1. (Optional) Fill in credentials in .env.local")
	fmt.Println("  2. Start the local stack:  nac up")
	fmt.Println("  3. Build workflows in n8n at http://localhost:5678")
	fmt.Println("  4. Export your work:      nac export workflows")
	fmt.Println("  5. Commit and push to deploy via CI")

	return nil
}

func createDotEnvLocal(dir string, force bool) error {
	dest := filepath.Join(dir, ".env.local")
	if _, err := os.Stat(dest); err == nil && !force {
		return nil // Already exists, don't overwrite
	}

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return fmt.Errorf("generating random key: %w", err)
	}
	randomKey := hex.EncodeToString(keyBytes)

	templateContent, err := templateFS.ReadFile("templates/env.local.example.tmpl")
	if err != nil {
		return fmt.Errorf("reading env template: %w", err)
	}

	content := strings.Replace(string(templateContent), "change-me-to-a-strong-random-key", randomKey, 1)

	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing .env.local: %w", err)
	}
	fmt.Printf("  created .env.local (with random N8N_ENCRYPTION_KEY)\n")
	return nil
}

// ... (rest of the file is the same)
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

	tmplContent, err := templateFS.ReadFile("templates/gitignore.tmpl")
	if err != nil {
		return fmt.Errorf("reading gitignore template: %w", err)
	}

	nacBlock := fmt.Sprintf("\n# === nac (n8n As Code) ===\n%s\n", string(tmplContent))

	if existing, err := os.ReadFile(gitignorePath); err == nil {
		if strings.Contains(string(existing), "# === nac (n8n As Code) ===") {
			return nil
		}
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(nacBlock)
		return err
	}

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
