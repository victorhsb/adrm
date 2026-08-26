package canon

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	exitOK       = 0
	exitUsage    = 2
	exitNotFound = 3
	exitState    = 4
	exitIO       = 5
)

// outputPayload is the capability every successful command payload
// implements: a deterministic text projection for --format text.
type outputPayload interface {
	renderText(io.Writer)
}

// contextPayload is the bounded Markdown projection available to
// --format context. Only the list payload implements it, matching the
// early supportsContextFormat gate in Run.
type contextPayload interface {
	renderContext(io.Writer)
}

// Renderer renders one envelope fully in a single output format.
type Renderer interface {
	Render(io.Writer, Envelope)
}

// renderers resolves validated format names to rendering strategies,
// replacing format-string dispatch in writeEnvelope.
var renderers = map[string]Renderer{
	"json":    jsonRenderer{},
	"text":    textRenderer{},
	"context": contextRenderer{},
}

// writeEnvelope applies envelope defaults and renders through the resolved
// renderer. Unknown formats render as JSON, preserving the historical
// fall-through for paths that write before format validation.
func writeEnvelope(out io.Writer, env Envelope, format string) {
	env.SchemaVersion = SchemaVersion
	if env.Status == "" {
		env.Status = "ok"
	}
	if renderer, ok := renderers[format]; ok {
		renderer.Render(out, env)
		return
	}
	jsonRenderer{}.Render(out, env)
}

// jsonRenderer owns the canon.v1 JSON contract: the versioned envelope,
// indented with two spaces.
type jsonRenderer struct{}

func (jsonRenderer) Render(out io.Writer, env Envelope) {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
}

// textRenderer writes the human opt-in projection: generic envelope lines
// (command, status, warnings, next actions) plus the payload's own text
// rendering. Payloads without a text projection cannot reach this path once
// they implement outputPayload, so data can no longer disappear silently.
type textRenderer struct{}

func (textRenderer) Render(out io.Writer, env Envelope) {
	if env.Error != nil {
		fmt.Fprintf(out, "error: %s\n%s\nsuggested fix: %s\n", env.Error.Code, env.Error.Message, env.Error.SuggestedFix)
		return
	}
	fmt.Fprintf(out, "%s: %s\n", env.Command, env.Status)
	if len(env.Warnings) > 0 {
		fmt.Fprintf(out, "warnings: %s\n", strings.Join(env.Warnings, "; "))
	}
	if payload, ok := env.Data.(outputPayload); ok {
		payload.renderText(out)
	}
	if len(env.NextActions) > 0 {
		fmt.Fprintln(out, "next actions:")
		for _, action := range env.NextActions {
			fmt.Fprintf(out, "- %s (%s): %s\n", action.Command, action.Safety, action.Description)
		}
	}
}

// contextRenderer writes the bounded Markdown projection for list payloads
// and the compact error block for context-formatted error envelopes.
type contextRenderer struct{}

func (contextRenderer) Render(out io.Writer, env Envelope) {
	if env.Error != nil {
		fmt.Fprintf(out, "## Canon Error\n\n- `%s`: %s\n", env.Error.Code, env.Error.Message)
		return
	}
	if payload, ok := env.Data.(contextPayload); ok {
		payload.renderContext(out)
	}
}

func errorEnvelope(command, code, category, message, suggestedFix string) Envelope {
	return Envelope{
		Command: command,
		Status:  "error",
		Error: &CLIError{
			Code:         code,
			Category:     category,
			Message:      message,
			SuggestedFix: suggestedFix,
		},
	}
}
