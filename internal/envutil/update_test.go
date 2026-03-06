package envutil

import (
	"os"
	"strings"
	"testing"
)

func TestUpdateEnvFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", ".env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	initial := `# comment
KEY1=val1
KEY2=val2 # inline comment
`
	if err := os.WriteFile(tmpfile.Name(), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	updates := map[string]string{
		"KEY1": "newval1",
		"KEY3": "val3",
	}

	if err := UpdateEnvFile(tmpfile.Name(), updates); err != nil {
		t.Fatalf("UpdateEnvFile failed: %v", err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "KEY1=newval1") {
		t.Errorf("KEY1 not updated correctly: %s", s)
	}
	if !strings.Contains(s, "KEY2=val2") {
		t.Errorf("KEY2 should be preserved: %s", s)
	}
	if !strings.Contains(s, "KEY3=val3") {
		t.Errorf("KEY3 not appended: %s", s)
	}
	if !strings.Contains(s, "# comment") {
		t.Errorf("Comments should be preserved: %s", s)
	}
}

func TestUpdateEnvFile_NewFile(t *testing.T) {
	filename := "nonexistent.env"
	defer os.Remove(filename)

	updates := map[string]string{
		"NEW": "val",
	}

	if err := UpdateEnvFile(filename, updates); err != nil {
		t.Fatalf("UpdateEnvFile failed: %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "NEW=val\n" {
		t.Errorf("Unexpected content: %q", string(content))
	}
}
