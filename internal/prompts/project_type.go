// Package prompts holds gitt's domain-level interactive prompts: which
// project type, which project, and similar create-time decisions shared by
// `gitt init`, `gitt clone`, and `gitt convert`. It composes the raw input
// primitives in internal/ui (Confirm, Select) with gitt-specific knowledge
// of which options exist, which are experimental, and how flag/--yes/prompt
// precedence resolves to a final value.
//
// Resolvers take primitive arguments (flag value, autoYes) instead of
// *cobra.Command so this package stays free of CLI framework dependencies
// and is unit-testable without spinning up a command. Each caller reads its
// own flags and forwards the values.
package prompts

import (
	"errors"
	"fmt"

	"github.com/foreverfl/gitt/internal/ui"
)

// Project types gitt understands. Only single-port is wired up end-to-end;
// multi-port is shown in prompts as a preview so users see what's coming,
// but selecting it is rejected until per-branch port allocation, compose
// overrides, etc. land.
const (
	ProjectTypeSinglePort = "single-port"
	ProjectTypeMultiPort  = "multi-port"
)

// ResolveProjectType decides which project type the user is committing to.
// Precedence: explicit flag value → autoYes default → interactive prompt.
// Multi-port is rejected wherever it surfaces.
//
// Callers pass flagValue from their --project-type flag (empty string when
// unset) and autoYes from the persistent --yes flag.
func ResolveProjectType(flagValue string, autoYes bool) (string, error) {
	if flagValue != "" {
		if flagValue != ProjectTypeSinglePort {
			return "", fmt.Errorf("project type %q not supported (multi-port is experimental, only %q is available)", flagValue, ProjectTypeSinglePort)
		}
		return flagValue, nil
	}
	if autoYes {
		return ProjectTypeSinglePort, nil
	}

	choice, err := ui.Select(
		"project type:",
		[]ui.Option{
			{Label: ProjectTypeSinglePort, Value: ProjectTypeSinglePort},
			{Label: ProjectTypeMultiPort, Value: ProjectTypeMultiPort, Disabled: true, Note: "experimental — not yet available"},
		},
		0,
	)
	if err != nil {
		if errors.Is(err, ui.ErrNoTTY) {
			return "", fmt.Errorf("non-interactive shell — pass --project-type %s or --yes", ProjectTypeSinglePort)
		}
		return "", err
	}
	return choice, nil
}
