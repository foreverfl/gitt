package prompts

import (
	"strings"
	"testing"
)

func TestResolveProjectType_FlagSinglePort(t *testing.T) {
	got, err := ResolveProjectType(ProjectTypeSinglePort, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ProjectTypeSinglePort {
		t.Errorf("got %q, want %q", got, ProjectTypeSinglePort)
	}
}

func TestResolveProjectType_FlagMultiPortRefused(t *testing.T) {
	_, err := ResolveProjectType(ProjectTypeMultiPort, false)
	if err == nil {
		t.Fatal("expected error when flag is multi-port")
	}
	if !strings.Contains(err.Error(), "multi-port") {
		t.Errorf("error = %v, want mention of multi-port", err)
	}
}

func TestResolveProjectType_FlagUnknownRefused(t *testing.T) {
	_, err := ResolveProjectType("triple-port", false)
	if err == nil {
		t.Fatal("expected error for unknown flag value")
	}
	if !strings.Contains(err.Error(), "triple-port") {
		t.Errorf("error = %v, want mention of unknown value", err)
	}
}

func TestResolveProjectType_AutoYesDefaults(t *testing.T) {
	got, err := ResolveProjectType("", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ProjectTypeSinglePort {
		t.Errorf("got %q, want %q", got, ProjectTypeSinglePort)
	}
}

func TestResolveProjectType_NonInteractiveErrors(t *testing.T) {
	// In `go test`, os.Stdin is not a TTY → ui.Select returns ErrNoTTY,
	// which the resolver wraps with a hint to pass --project-type or --yes.
	_, err := ResolveProjectType("", false)
	if err == nil {
		t.Fatal("expected error in non-interactive shell")
	}
	if !strings.Contains(err.Error(), "non-interactive") {
		t.Errorf("error = %v, want mention of non-interactive", err)
	}
}
