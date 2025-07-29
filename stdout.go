// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"io"
	"os"
)

type stdoutContextKey int

var stdoutKey stdoutContextKey

// WithStdout adds the input io.Writer to the input context. Command functions
// that wish to support different stdout backends can retrieve the io.Writer
// from the context.
func WithStdout(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, stdoutKey, w)
}

// Stdout retrieves io.Writer meant to capture stdout from the input
// context. Returns os.Stdout when no such io.Writer is found.
func Stdout(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stdoutKey).(io.Writer); ok {
		return w
	}
	return os.Stdout
}
