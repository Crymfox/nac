package envutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// UpdateEnvFile updates or appends key-value pairs in a .env file.
// It preserves existing comments and formatting where possible.
func UpdateEnvFile(filename string, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}

	var lines []string
	foundKeys := make(map[string]bool)

	// Read existing file if it exists
	file, err := os.Open(filename)
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Skip empty lines or comments
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				lines = append(lines, line)
				continue
			}

			// Check if it's a KEY=VALUE line
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				if newValue, ok := updates[key]; ok {
					// Update existing line
					lines = append(lines, fmt.Sprintf("%s=%s", key, newValue))
					foundKeys[key] = true
					continue
				}
			}
			lines = append(lines, line)
		}
		file.Close()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("opening %s: %w", filename, err)
	}

	// Add missing keys at the end
	addedAny := false
	for key, val := range updates {
		if !foundKeys[key] {
			if !addedAny && len(lines) > 0 && lines[len(lines)-1] != "" {
				lines = append(lines, "") // Add a newline before the new block
			}
			lines = append(lines, fmt.Sprintf("%s=%s", key, val))
			addedAny = true
		}
	}

	return os.WriteFile(filename, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
