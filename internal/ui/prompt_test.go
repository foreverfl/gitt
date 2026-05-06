package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
		wantErr    error
	}{
		{name: "empty defaults to yes", input: "\n", defaultYes: true, want: true},
		{name: "empty defaults to no", input: "\n", defaultYes: false, want: false},
		{name: "y", input: "y\n", want: true},
		{name: "yes", input: "yes\n", want: true},
		{name: "uppercase YES", input: "YES\n", want: true},
		{name: "n", input: "n\n", defaultYes: true, want: false},
		{name: "no", input: "no\n", defaultYes: true, want: false},
		{name: "trims whitespace", input: "  y  \n", want: true},
		{name: "retries past garbage", input: "wat\ny\n", want: true},
		{name: "three invalid responses errors", input: "a\nb\nc\n"},
		{name: "stdin closed before answer", input: "", wantErr: ErrNoTTY},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			got, err := confirm(strings.NewReader(tc.input), out, "?", tc.defaultYes)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if tc.name == "three invalid responses errors" {
				if err == nil {
					t.Fatalf("expected error after %d invalid responses", maxAttempts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSelect(t *testing.T) {
	options := []Option{
		{Label: "single-port", Value: "single-port"},
		{Label: "multi-port", Value: "multi-port", Disabled: true, Note: "experimental"},
	}

	cases := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "empty picks default", input: "\n", want: "single-port"},
		{name: "by number", input: "1\n", want: "single-port"},
		{name: "by label", input: "single-port\n", want: "single-port"},
		{name: "case insensitive label", input: "Single-Port\n", want: "single-port"},
		{name: "disabled retried then valid", input: "2\n1\n", want: "single-port"},
		{name: "out of range retried then valid", input: "9\n1\n", want: "single-port"},
		{name: "garbage retried then valid", input: "wat\n1\n", want: "single-port"},
		{name: "too many bad responses", input: "9\n9\n9\n"},
		{name: "stdin closed", input: "", wantErr: ErrNoTTY},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			got, err := selectFromOptions(strings.NewReader(tc.input), out, "pick:", options, 0)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if tc.name == "too many bad responses" {
				if err == nil {
					t.Fatalf("expected error after %d invalid responses", maxAttempts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSelectRendersDisabledMarker(t *testing.T) {
	options := []Option{
		{Label: "single-port", Value: "single-port"},
		{Label: "multi-port", Value: "multi-port", Disabled: true, Note: "experimental"},
	}
	out := &bytes.Buffer{}
	if _, err := selectFromOptions(strings.NewReader("\n"), out, "pick:", options, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "multi-port") {
		t.Errorf("output missing multi-port label: %q", rendered)
	}
	if !strings.Contains(rendered, "experimental") {
		t.Errorf("output missing experimental note: %q", rendered)
	}
	if !strings.Contains(rendered, "(unavailable)") {
		t.Errorf("output missing (unavailable) marker: %q", rendered)
	}
}

func TestSelectRejectsBadDefaults(t *testing.T) {
	options := []Option{
		{Label: "a", Value: "a"},
		{Label: "b", Value: "b", Disabled: true},
	}
	out := &bytes.Buffer{}
	if _, err := selectFromOptions(strings.NewReader("\n"), out, "?", options, 5); err == nil {
		t.Errorf("expected error for out-of-range default")
	}
	if _, err := selectFromOptions(strings.NewReader("\n"), out, "?", options, 1); err == nil {
		t.Errorf("expected error for disabled default")
	}
	if _, err := selectFromOptions(strings.NewReader("\n"), out, "?", nil, 0); err == nil {
		t.Errorf("expected error for empty options")
	}
}

func TestConfirmHintReflectsDefault(t *testing.T) {
	cases := []struct {
		defaultYes bool
		wantHint   string
	}{
		{defaultYes: true, wantHint: "[Y/n]"},
		{defaultYes: false, wantHint: "[y/N]"},
	}
	for _, tc := range cases {
		out := &bytes.Buffer{}
		_, err := confirm(strings.NewReader("\n"), out, "proceed?", tc.defaultYes)
		if err != nil {
			t.Fatalf("defaultYes=%v: unexpected error: %v", tc.defaultYes, err)
		}
		if !strings.Contains(out.String(), tc.wantHint) {
			t.Errorf("defaultYes=%v: prompt %q missing hint %q", tc.defaultYes, out.String(), tc.wantHint)
		}
	}
}