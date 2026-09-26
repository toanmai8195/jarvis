package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMessage(t *testing.T) {
	tests := []struct {
		name      string
		component string
		want      string
	}{
		{name: "empty defaults to smoke", component: "", want: "snaptix smoke ok"},
		{name: "whitespace only defaults to smoke", component: " \t\n", want: "snaptix smoke ok"},
		{name: "named component", component: "core", want: "snaptix core ok"},
		{name: "trims surrounding spaces", component: "  stats-worker ", want: "snaptix stats-worker ok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := message(tt.component); got != tt.want {
				t.Errorf("message(%q) = %q, want %q", tt.component, got, tt.want)
			}
		})
	}
}

// failWriter luôn trả lỗi, để kiểm nhánh lỗi của run.
type failWriter struct{ err error }

func (f failWriter) Write([]byte) (int, error) { return 0, f.err }

func TestRun(t *testing.T) {
	errBroken := errors.New("broken pipe")
	tests := []struct {
		name    string
		w       io.Writer
		wantOut string
		wantErr error
	}{
		{name: "writes exactly one line", w: &strings.Builder{}, wantOut: "snaptix smoke ok\n"},
		{name: "propagates writer error", w: failWriter{err: errBroken}, wantErr: errBroken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.w)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, want %v", err, tt.wantErr)
			}
			if sb, ok := tt.w.(*strings.Builder); ok && sb.String() != tt.wantOut {
				t.Errorf("run() wrote %q, want %q", sb.String(), tt.wantOut)
			}
		})
	}
}
