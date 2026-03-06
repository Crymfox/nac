package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ComposeUp runs docker compose up -d
func ComposeUp(composeFile string) error {
	args := []string{"compose", "-f", composeFile}
	if _, err := os.Stat(".env.local"); err == nil {
		args = append(args, "--env-file", ".env.local")
	}
	args = append(args, "up", "-d")

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}
	return nil
}

// ComposeDown runs docker compose down
func ComposeDown(composeFile string) error {
	args := []string{"compose", "-f", composeFile}
	if _, err := os.Stat(".env.local"); err == nil {
		args = append(args, "--env-file", ".env.local")
	}
	args = append(args, "down")

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}
	return nil
}

// ComposeLogs runs docker compose logs -f
func ComposeLogs(composeFile string, service string) error {
	args := []string{"compose", "-f", composeFile}
	if _, err := os.Stat(".env.local"); err == nil {
		args = append(args, "--env-file", ".env.local")
	}
	args = append(args, "logs", "-f")
	if service != "" {
		args = append(args, service)
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DetectNetwork auto-detects the n8n Docker network.
// This is useful for running local tools that need to talk to the DB.
func DetectNetwork() (string, error) {
	// Find postgres container
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("listing containers: %w", err)
	}

	var pgContainer string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "postgres") && strings.Contains(line, "n8n") {
			pgContainer = line
			break
		}
	}

	if pgContainer == "" {
		// Try fallback
		cmd = exec.Command("docker", "network", "ls", "--format", "{{.Name}}")
		out, err = cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "n8n") {
					return line, nil
				}
			}
		}
		return "n8n_default", nil
	}

	// Inspect container to get its network
	cmd = exec.Command("docker", "inspect", "-f", "{{range $net,$v := .NetworkSettings.Networks}}{{$net}}{{end}}", pgContainer)
	out, err = cmd.Output()
	if err != nil {
		return "n8n_default", nil
	}

	network := strings.TrimSpace(string(out))
	if network != "" {
		return network, nil
	}

	return "n8n_default", nil
}
