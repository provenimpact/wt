// Feature: worktree-management
// Spec version: 1.0.0
// Generated from: spec.adoc
//
// Spec coverage:
//   WT-026: Shell function for directory change
//   WT-027: Shell init command outputs function code
//   WT-028: Support Bash, Zsh, and Fish

package shell

import (
	"strings"
	"testing"
)

// WT-027: When the user invokes `wt init <shell>`, the system shall output
// the shell function code for the specified shell.
// WT-028: The system shall support shell integration for Bash, Zsh, and Fish shells.
func TestGenerate_SupportedShells(t *testing.T) {
	tests := []struct {
		shell string
	}{
		{"bash"},
		{"zsh"},
		{"fish"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			code, err := Generate(tt.shell)
			if err != nil {
				t.Fatalf("Generate(%q) returned error: %v", tt.shell, err)
			}
			if code == "" {
				t.Fatalf("Generate(%q) returned empty string", tt.shell)
			}
		})
	}
}

// WT-026: The system shall provide a shell function that wraps the wt binary,
// enabling directory change to the selected worktree.
func TestGenerate_BashContainsCdLogic(t *testing.T) {
	code, err := Generate("bash")
	if err != nil {
		t.Fatal(err)
	}

	// Must define a function called wt
	if !strings.Contains(code, "wt()") {
		t.Error("bash output does not define wt() function")
	}
	// Must check for __wt_cd: sentinel
	if !strings.Contains(code, "__wt_cd:") {
		t.Error("bash output does not check for __wt_cd: sentinel")
	}
	// Must call cd
	if !strings.Contains(code, "cd ") {
		t.Error("bash output does not contain cd command")
	}
	// Must call the real binary via `command wt`
	if !strings.Contains(code, "command wt") {
		t.Error("bash output does not call `command wt`")
	}
}

func TestGenerate_ZshSameAsBash(t *testing.T) {
	bash, _ := Generate("bash")
	zsh, _ := Generate("zsh")
	if bash != zsh {
		t.Error("bash and zsh should produce identical output")
	}
}

func TestGenerate_FishContainsCdLogic(t *testing.T) {
	code, err := Generate("fish")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "function wt") {
		t.Error("fish output does not define wt function")
	}
	if !strings.Contains(code, "__wt_cd:") {
		t.Error("fish output does not check for __wt_cd: sentinel")
	}
	if !strings.Contains(code, "cd ") {
		t.Error("fish output does not contain cd command")
	}
	if !strings.Contains(code, "command wt") {
		t.Error("fish output does not call `command wt`")
	}
}

func TestGenerate_UnsupportedShell(t *testing.T) {
	_, err := Generate("powershell")
	if err == nil {
		t.Error("Generate(\"powershell\") should return error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention 'unsupported', got: %v", err)
	}
}
