// Package banner renders the doctree welcome banner shown on `doctree on`.
package banner

import (
	_ "embed"
	"fmt"
	"io"
	"strings"
)

//go:embed art.txt
var artText string

const (
	skyBlue = "\033[38;5;117m"
	reset   = "\033[0m"
)

// Print writes the doctree welcome banner to out. version is shown centered
// below the art; pass an empty string if unknown.
func Print(out io.Writer, version string) {
	art := strings.Split(strings.TrimRight(artText, "\n"), "\n")

	width := 0
	for _, line := range art {
		if count := runeCount(line); count > width {
			width = count
		}
	}

	label := "doctree"
	if version != "" {
		label = "doctree " + version
	}
	if count := runeCount(label); count > width {
		width = count
	}

	dashes := strings.Repeat("─", width+2)
	fmt.Fprintln(out, skyBlue+"╭"+dashes+"╮"+reset)
	fmt.Fprintln(out, row("", width))
	for _, line := range art {
		fmt.Fprintln(out, row(line, width))
	}
	fmt.Fprintln(out, row("", width))
	fmt.Fprintln(out, row(centered(label, width), width))
	fmt.Fprintln(out, row("", width))
	fmt.Fprintln(out, skyBlue+"╰"+dashes+"╯"+reset)
}

// row renders one inner row: sky-blue side borders, single-space inner
// padding, content padded with spaces to width visual cells. Assumes every
// rune in content is single-width.
func row(content string, width int) string {
	pad := width - runeCount(content)
	if pad < 0 {
		pad = 0
	}
	return skyBlue + "│" + reset +
		" " + content + strings.Repeat(" ", pad) + " " +
		skyBlue + "│" + reset
}

func centered(text string, width int) string {
	count := runeCount(text)
	if count >= width {
		return text
	}
	return strings.Repeat(" ", (width-count)/2) + text
}

func runeCount(text string) int {
	count := 0
	for range text {
		count++
	}
	return count
}
