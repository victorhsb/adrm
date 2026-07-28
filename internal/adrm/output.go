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
	renderDataText(out, env.Command, env.Data)
	if len(env.NextActions) > 0 {
		fmt.Fprintln(out, "next actions:")
		for _, action := range env.NextActions {
			fmt.Fprintf(out, "- %s (%s): %s\n", action.Command, action.Safety, action.Description)
		}
	}
}

func renderDataText(out io.Writer, command string, data any) {
	if data == nil {
		return
	}
	payload, ok := data.(map[string]any)
	if !ok {
		return
	}
	switch command {
	case "commands":
		renderCommandsText(out, payload)
	case "doctor":
		renderDoctorText(out, payload)
	case "list":
		renderListText(out, payload)
	case "show":
		renderShowText(out, payload)
	case "search":
		renderSearchText(out, payload)
	case "init", "new", "accept", "reject", "supersede", "deprecate", "append", "skill install", "skill update":
		renderMutationText(out, payload)
	case "skill":
		renderSkillText(out, payload)
	}
}

func jsonCopy(src, dst any) bool {
	b, err := json.Marshal(src)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, dst) == nil
}

func renderCommandsText(out io.Writer, payload map[string]any) {
	var info struct {
		Commands    []CommandInfo       `json:"commands"`
		GlobalFlags []map[string]string `json:"global_flags"`
	}
	if !jsonCopy(payload, &info) {
		return
	}
	fmt.Fprintln(out, "commands:")
	for _, cmd := range info.Commands {
		fmt.Fprintf(out, "  %s\n", cmd.Name)
		fmt.Fprintf(out, "    purpose: %s\n", cmd.Purpose)
		fmt.Fprintf(out, "    safety: %s\n", cmd.Safety)
		if cmd.Mutating {
			fmt.Fprintln(out, "    mutating: true")
		}
		if cmd.HasDryRun {
			fmt.Fprintln(out, "    has_dry_run: true")
		}
		if len(cmd.Selectors) > 0 {
			fmt.Fprintf(out, "    selectors: %s\n", strings.Join(cmd.Selectors, ", "))
		}
		for _, example := range cmd.Examples {
			fmt.Fprintf(out, "    example: %s\n", example)
		}
	}
	if len(info.GlobalFlags) > 0 {
		fmt.Fprintln(out, "global flags:")
		for _, flag := range info.GlobalFlags {
			fmt.Fprintf(out, "  %s (default: %s)\n    %s\n", flag["name"], flag["default"], flag["purpose"])
		}
	}
}

func renderDoctorText(out io.Writer, payload map[string]any) {
	var data struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if !jsonCopy(payload, &data) {
		return
	}
	fmt.Fprintln(out, "diagnostics:")
	for _, d := range data.Diagnostics {
		fmt.Fprintf(out, "  %s: %s - %s\n", d.Name, d.Status, d.Message)
	}
}

func renderListText(out io.Writer, payload map[string]any) {
	var data struct {
		Count int   `json:"count"`
		ADRs  []ADR `json:"adrs"`
	}
	if !jsonCopy(payload, &data) {
		return
	}
	fmt.Fprintf(out, "count: %d\n", data.Count)
	if len(data.ADRs) > 0 {
		fmt.Fprintln(out, "adrs:")
		for _, adr := range data.ADRs {
			tags := strings.Join(adr.Tags, ", ")
			if tags == "" {
				tags = "-"
			}
			fmt.Fprintf(out, "  %s: %s [%s] (%s)\n", adr.ID, adr.Title, adr.Status, tags)
		}
	}
}
func renderShowText(out io.Writer, payload map[string]any) {
	var data struct {
		ADR ADR `json:"adr"`
	}
	if !jsonCopy(payload, &data) {
		return
	}
	renderADRText(out, data.ADR, true)
}

func renderSearchText(out io.Writer, payload map[string]any) {
	var data struct {
		Query   string           `json:"query"`
		Count   int              `json:"count"`
		Results []map[string]any `json:"results"`
	}
	if !jsonCopy(payload, &data) {
		return
	}
	fmt.Fprintf(out, "query: %s\n", data.Query)
	fmt.Fprintf(out, "count: %d\n", data.Count)
	if len(data.Results) > 0 {
		fmt.Fprintln(out, "results:")
		for _, r := range data.Results {
			var adr ADR
			var snippet string
			if a, ok := r["adr"]; ok {
				_ = jsonCopy(a, &adr)
			}
			if s, ok := r["snippet"].(string); ok {
				snippet = s
			}
			fmt.Fprintf(out, "  %s: %s [%s]\n", adr.ID, adr.Title, adr.Status)
			if snippet != "" {
				fmt.Fprintf(out, "    snippet: %s\n", snippet)
			}
		}
	}
}

func renderMutationText(out io.Writer, payload map[string]any) {
	var data struct {
		Plan Plan `json:"plan"`
		ADR  ADR  `json:"adr"`
	}
	if !jsonCopy(payload, &data) {
		return
	}
	if len(data.Plan.Operations) > 0 {
		fmt.Fprintln(out, "plan:")
		for _, op := range data.Plan.Operations {
			fmt.Fprintf(out, "  %s: %s\n    %s\n", op.Action, op.Path, op.Description)
		}
		fmt.Fprintf(out, "dry_run: %t\n", data.Plan.DryRun)
		fmt.Fprintf(out, "changes_made: %t\n", data.Plan.ChangesMade)
	}
	if data.ADR.ID != "" {
		renderADRText(out, data.ADR, false)
	}
}

func renderSkillText(out io.Writer, payload map[string]any) {
	var data struct {
		Name    string         `json:"filename"`
		Content string         `json:"content"`
		Skill   map[string]any `json:"skill"`
	}
	if !jsonCopy(payload, &data) {
		return
	}
	fmt.Fprintf(out, "filename: %s\n", data.Name)
	if name, ok := data.Skill["name"].(string); ok {
		fmt.Fprintf(out, "skill: %s\n", name)
	}
	if version, ok := data.Skill["version"].(string); ok {
		fmt.Fprintf(out, "version: %s\n", version)
	}
	if hash, ok := data.Skill["hash"].(string); ok {
		fmt.Fprintf(out, "hash: %s\n", hash)
	}
	if data.Content != "" {
		fmt.Fprintln(out, "---")
		fmt.Fprintln(out, data.Content)
	}
}

func renderADRText(out io.Writer, adr ADR, includeContent bool) {
	fmt.Fprintln(out, "adr:")
	if adr.Kind != "" {
		fmt.Fprintf(out, "  kind: %s\n", adr.Kind)
	}
	fmt.Fprintf(out, "  id: %s\n", adr.ID)
	fmt.Fprintf(out, "  title: %s\n", adr.Title)
	fmt.Fprintf(out, "  status: %s\n", adr.Status)
	fmt.Fprintf(out, "  date: %s\n", adr.Date)
	if len(adr.Tags) > 0 {
		fmt.Fprintf(out, "  tags: %s\n", strings.Join(adr.Tags, ", "))
	}
	if adr.SupersededBy != "" {
		fmt.Fprintf(out, "  superseded_by: %s\n", adr.SupersededBy)
	}
	if len(adr.Supersedes) > 0 {
		fmt.Fprintf(out, "  supersedes: %s\n", strings.Join(adr.Supersedes, ", "))
	}
	if adr.DeprecatedBy != "" {
		fmt.Fprintf(out, "  deprecated_by: %s\n", adr.DeprecatedBy)
	}
	fmt.Fprintf(out, "  path: %s\n", adr.Path)
	if includeContent && adr.Content != "" {
		fmt.Fprintln(out, "  content:")
		for _, line := range strings.Split(adr.Content, "\n") {
			fmt.Fprintf(out, "    %s\n", line)
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
