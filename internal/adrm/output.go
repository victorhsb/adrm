package adrm

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

func writeEnvelope(out io.Writer, env Envelope, format string) {
	env.SchemaVersion = SchemaVersion
	if env.Status == "" {
		env.Status = "ok"
	}
	if format == "text" {
		writeText(out, env)
		return
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
}

func writeText(out io.Writer, env Envelope) {
	if env.Error != nil {
		fmt.Fprintf(out, "error: %s\n%s\nsuggested fix: %s\n", env.Error.Code, env.Error.Message, env.Error.SuggestedFix)
		return
	}
	fmt.Fprintf(out, "%s: %s\n", env.Command, env.Status)
	if len(env.Warnings) > 0 {
		fmt.Fprintf(out, "warnings: %s\n", strings.Join(env.Warnings, "; "))
	}
	if len(env.NextActions) > 0 {
		fmt.Fprintln(out, "next actions:")
		for _, action := range env.NextActions {
			fmt.Fprintf(out, "- %s (%s): %s\n", action.Command, action.Safety, action.Description)
		}
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
