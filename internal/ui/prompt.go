package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ErrNoTTY signals that stdin is not a terminal, so an interactive prompt
// would block on a closed/piped stream. Callers should translate this into
// a user-facing error like "use --yes to bypass confirmation".
var ErrNoTTY = errors.New("stdin is not a terminal")

const maxAttempts = 3

// Confirm asks a yes/no question on stdin/stderr and returns the user's
// answer. defaultYes controls both the [Y/n] vs [y/N] hint and the value
// returned when the user just hits enter.
func Confirm(message string, defaultYes bool) (bool, error) {
	return confirm(os.Stdin, os.Stderr, message, defaultYes)
}

func confirm(in io.Reader, out io.Writer, message string, defaultYes bool) (bool, error) {
	if file, ok := in.(*os.File); ok {
		stat, err := file.Stat()
		if err != nil {
			return false, err
		}
		if stat.Mode()&os.ModeCharDevice == 0 {
			return false, ErrNoTTY
		}
	}

	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	reader := bufio.NewReader(in)
	for range maxAttempts {
		if _, err := fmt.Fprintf(out, "%s %s ", message, hint); err != nil {
			return false, err
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, ErrNoTTY
			}
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "":
			return defaultYes, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Fprintln(out, "please answer y or n")
	}
	return false, fmt.Errorf("too many invalid responses")
}

// Option is one entry in a Select prompt. Disabled options are still rendered
// (so users can see what's coming) but cannot be picked — choosing one
// re-prompts with the option's Note as the rejection message.
type Option struct {
	Label    string
	Value    string
	Disabled bool
	Note     string
}

// Select asks the user to pick one of the given options on stdin/stderr and
// returns the chosen option's Value. The user may type either the option
// number (1-based) or the option's label; an empty line picks defaultIndex.
func Select(message string, options []Option, defaultIndex int) (string, error) {
	return selectFromOptions(os.Stdin, os.Stderr, message, options, defaultIndex)
}

func selectFromOptions(in io.Reader, out io.Writer, message string, options []Option, defaultIndex int) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		return "", fmt.Errorf("default index %d out of range", defaultIndex)
	}
	if options[defaultIndex].Disabled {
		return "", fmt.Errorf("default option %q is disabled", options[defaultIndex].Label)
	}

	if file, ok := in.(*os.File); ok {
		stat, err := file.Stat()
		if err != nil {
			return "", err
		}
		if stat.Mode()&os.ModeCharDevice == 0 {
			return "", ErrNoTTY
		}
	}

	reader := bufio.NewReader(in)
	for range maxAttempts {
		if _, err := fmt.Fprintln(out, message); err != nil {
			return "", err
		}
		for index, option := range options {
			marker := " "
			if index == defaultIndex {
				marker = "*"
			}
			suffix := ""
			if option.Note != "" {
				suffix = " — " + option.Note
			}
			if option.Disabled {
				suffix += " (unavailable)"
			}
			if _, err := fmt.Fprintf(out, "  %s %d) %s%s\n", marker, index+1, option.Label, suffix); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(out, "choose [1-%d, default %d]: ", len(options), defaultIndex+1); err != nil {
			return "", err
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", ErrNoTTY
			}
			return "", err
		}
		answer := strings.TrimSpace(line)
		if answer == "" {
			return options[defaultIndex].Value, nil
		}

		picked := -1
		if number, err := strconv.Atoi(answer); err == nil && number >= 1 && number <= len(options) {
			picked = number - 1
		} else {
			lower := strings.ToLower(answer)
			for index, option := range options {
				if strings.ToLower(option.Label) == lower {
					picked = index
					break
				}
			}
		}
		if picked < 0 {
			fmt.Fprintf(out, "please enter a number from 1 to %d, or an option label\n", len(options))
			continue
		}
		if options[picked].Disabled {
			message := options[picked].Note
			if message == "" {
				message = "not available yet"
			}
			fmt.Fprintf(out, "%q is %s — pick another\n", options[picked].Label, message)
			continue
		}
		return options[picked].Value, nil
	}
	return "", fmt.Errorf("too many invalid responses")
}
