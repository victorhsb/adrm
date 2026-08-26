package canon

import (
	"fmt"
	"io"
	"strings"

	"github.com/victorhsb/canon/skill"
)

// This file defines one typed struct for each stable command payload. Field
// order matters: envelopes used to carry map[string]any values, which Go's
// JSON encoder emits with sorted keys, so struct fields are declared in the
// encoded order of the maps they replace. Payloads that reuse existing types
// (ADR, Plan, Diagnostic, CommandInfo, configReport, skill.CatalogAsset)
// inherit the byte shape those types already produce.
//
// Every successful payload implements outputPayload, so text output can never
// drop data because a command name was missing from a dispatch switch. Only
// listPayload implements contextPayload; the early context gate in Run keeps
// those checks aligned.

var (
	_ outputPayload = rootPayload{}
	_ outputPayload = commandsPayload{}
	_ outputPayload = versionPayload{}
	_ outputPayload = doctorPayload{}
	_ outputPayload = validatePayload{}
	_ outputPayload = configReport{}
	_ outputPayload = configValidatePayload{}
	_ outputPayload = listPayload{}
	_ outputPayload = showPayload{}
	_ outputPayload = searchPayload{}
	_ outputPayload = mutationPayload{}
	_ outputPayload = planDryRunPayload{}
	_ outputPayload = Plan{}
	_ outputPayload = skillCatalogPayload{}
	_ outputPayload = skillMutationPayload{}

	_ contextPayload = listPayload{}
)

// rootPayload is the bare `canon` response: purpose plus command inventory.
type rootPayload struct {
	Commands []string `json:"commands"`
	Purpose  string   `json:"purpose"`
}

func (p rootPayload) renderText(out io.Writer) {
	fmt.Fprintf(out, "purpose: %s\n", p.Purpose)
	if len(p.Commands) > 0 {
		fmt.Fprintln(out, "commands:")
		for _, name := range p.Commands {
			fmt.Fprintf(out, "  %s\n", name)
		}
	}
}

// globalFlag describes one global CLI flag in the commands payload.
type globalFlag struct {
	Default string `json:"default"`
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
}

type commandsPayload struct {
	Commands    []CommandInfo `json:"commands"`
	GlobalFlags []globalFlag  `json:"global_flags"`
}

func (p commandsPayload) renderText(out io.Writer) {
	fmt.Fprintln(out, "commands:")
	for _, cmd := range p.Commands {
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
	if len(p.GlobalFlags) > 0 {
		fmt.Fprintln(out, "global flags:")
		for _, flag := range p.GlobalFlags {
			fmt.Fprintf(out, "  %s (default: %s)\n    %s\n", flag.Name, flag.Default, flag.Purpose)
		}
	}
}

type versionPayload struct {
	Version string `json:"version"`
}

func (p versionPayload) renderText(out io.Writer) {
	fmt.Fprintf(out, "version: %s\n", p.Version)
}

type doctorPayload struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

func (p doctorPayload) renderText(out io.Writer) {
	fmt.Fprintln(out, "diagnostics:")
	for _, d := range p.Diagnostics {
		fmt.Fprintf(out, "  %s: %s - %s\n", d.Name, d.Status, d.Message)
	}
}

// validatePayload is the shared payload for `canon validate` and the
// kind-scoped validate commands.
type validatePayload struct {
	Findings []Diagnostic      `json:"findings"`
	Summary  validationSummary `json:"summary"`
}

func (p validatePayload) renderText(out io.Writer) {
	fmt.Fprintln(out, "findings:")
	for _, f := range p.Findings {
		where := f.Path
		if f.ID != "" {
			where = strings.TrimSpace(where + " " + f.ID)
		}
		if where != "" {
			fmt.Fprintf(out, "  %s: %s - %s (%s)\n", f.Name, f.Status, f.Message, where)
			continue
		}
		fmt.Fprintf(out, "  %s: %s - %s\n", f.Name, f.Status, f.Message)
	}
	fmt.Fprintf(out, "summary: files_checked=%d errors=%d warnings=%d\n", p.Summary.FilesChecked, p.Summary.Errors, p.Summary.Warnings)
}

// configValidatePayload extends the validation payload with the loaded
// configuration report. The config key is omitted when no single file
// resolved, so Config stays a pointer with omitempty.
type configValidatePayload struct {
	Config   *configReport     `json:"config,omitempty"`
	Findings []Diagnostic      `json:"findings"`
	Summary  validationSummary `json:"summary"`
}

func (p configValidatePayload) renderText(out io.Writer) {
	validatePayload{Findings: p.Findings, Summary: p.Summary}.renderText(out)
	if p.Config != nil {
		p.Config.renderText(out)
	}
}

// listPayload carries document summaries for the combined and kind-scoped
// list commands. scope names the kind prefix ("" for the combined list) and
// drives the context heading; it is never encoded.
type listPayload struct {
	ADRs  []ADR  `json:"adrs"`
	Count int    `json:"count"`
	scope string `json:"-"`
}

func (p listPayload) renderText(out io.Writer) {
	fmt.Fprintf(out, "count: %d\n", p.Count)
	if len(p.ADRs) > 0 {
		fmt.Fprintln(out, "adrs:")
		for _, adr := range p.ADRs {
			tags := strings.Join(adr.Tags, ", ")
			if tags == "" {
				tags = "-"
			}
			fmt.Fprintf(out, "  %s: %s [%s] (%s)\n", adr.ID, adr.Title, adr.Status, tags)
		}
	}
}

func (p listPayload) renderContext(out io.Writer) {
	fmt.Fprintf(out, "## %s\n\n", p.contextHeading())
	if len(p.ADRs) == 0 {
		fmt.Fprintln(out, "_No matching documents._")
		return
	}
	for _, adr := range p.ADRs {
		fmt.Fprintf(out, "- `%s`: %s\n", adr.ID, adr.Title)
	}
}

func (p listPayload) contextHeading() string {
	switch p.scope {
	case KindADR:
		return "Architecture Decision Records"
	case KindSPEC:
		return "Specifications"
	case KindDomain:
		return "Domain Model"
	default:
		return "Project Documents"
	}
}

type showPayload struct {
	ADR ADR `json:"adr"`
}

func (p showPayload) renderText(out io.Writer) {
	renderDocumentText(out, p.ADR, true)
}

// searchResult pairs one matching document with its snippet. The ADR JSON
// name is pinned by ADR-0011 across all kinds.
type searchResult struct {
	ADR     ADR    `json:"adr"`
	Snippet string `json:"snippet"`
}

type searchPayload struct {
	Count   int            `json:"count"`
	Query   string         `json:"query"`
	Results []searchResult `json:"results"`
}

func (p searchPayload) renderText(out io.Writer) {
	fmt.Fprintf(out, "query: %s\n", p.Query)
	fmt.Fprintf(out, "count: %d\n", p.Count)
	if len(p.Results) > 0 {
		fmt.Fprintln(out, "results:")
		for _, r := range p.Results {
			fmt.Fprintf(out, "  %s: %s [%s]\n", r.ADR.ID, r.ADR.Title, r.ADR.Status)
			if r.Snippet != "" {
				fmt.Fprintf(out, "    snippet: %s\n", r.Snippet)
			}
		}
	}
}

// mutationPayload is the applied or previewed single-document mutation
// envelope: lifecycle transitions, append, and `new` in both modes.
type mutationPayload struct {
	ADR  ADR  `json:"adr"`
	Plan Plan `json:"plan"`
}

func (p mutationPayload) renderText(out io.Writer) {
	renderPlanText(out, p.Plan)
	if p.ADR.ID != "" {
		renderDocumentText(out, p.ADR, false)
	}
}

// planDryRunPayload is the lifecycle dry-run envelope: the plan plus the
// targeted document id.
type planDryRunPayload struct {
	Plan     Plan   `json:"plan"`
	TargetID string `json:"target_id"`
}

func (p planDryRunPayload) renderText(out io.Writer) {
	renderPlanText(out, p.Plan)
}

// renderText makes a bare Plan text-capable so `adr init`, `spec init`, and
// `domain init` render their preview instead of silently omitting it.
func (p Plan) renderText(out io.Writer) {
	renderPlanText(out, p)
}

type skillCatalogPayload struct {
	Assets          []skill.CatalogAsset `json:"assets"`
	DefaultSkillDir string               `json:"default_skill_dir"`
}

func (p skillCatalogPayload) renderText(out io.Writer) {
	fmt.Fprintf(out, "default_skill_dir: %s\n", p.DefaultSkillDir)
	fmt.Fprintln(out, "assets:")
	for _, asset := range p.Assets {
		fmt.Fprintf(out, "  %s [%s]\n", asset.Name, asset.Kind)
		fmt.Fprintf(out, "    version: %s\n", asset.Version)
		fmt.Fprintf(out, "    hash: %s\n", asset.Hash)
		if len(asset.TargetPaths) > 0 {
			fmt.Fprintln(out, "    target paths:")
			for _, path := range asset.TargetPaths {
				fmt.Fprintf(out, "      %s\n", path)
			}
		}
	}
}

// skillManagedAsset describes one selected bundle asset in install and update
// envelopes. The key order reproduces the previous map encoding exactly;
// skill.CatalogAsset keeps its own declaration order for the catalog command,
// which is why the two shapes cannot share a struct.
type skillManagedAsset struct {
	Hash        string   `json:"hash"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	TargetPaths []string `json:"target_paths"`
	Version     string   `json:"version"`
}

// skillMutationPayload is the install and update envelope, in both preview
// and applied forms.
type skillMutationPayload struct {
	Assets  []skillManagedAsset `json:"assets"`
	Plan    Plan                `json:"plan"`
	Targets []string            `json:"targets"`
}

func (p skillMutationPayload) renderText(out io.Writer) {
	renderPlanText(out, p.Plan)
}

// renderPlanText renders the shared plan projection used by every mutation
// payload. An empty operation list renders nothing, matching the historical
// behavior where plans without operations were skipped.
func renderPlanText(out io.Writer, plan Plan) {
	if len(plan.Operations) == 0 {
		return
	}
	fmt.Fprintln(out, "plan:")
	for _, op := range plan.Operations {
		fmt.Fprintf(out, "  %s: %s\n    %s\n", op.Action, op.Path, op.Description)
	}
	fmt.Fprintf(out, "dry_run: %t\n", plan.DryRun)
	fmt.Fprintf(out, "changes_made: %t\n", plan.ChangesMade)
}

func renderDocumentText(out io.Writer, doc ADR, includeContent bool) {
	fmt.Fprintln(out, "adr:")
	if doc.Kind != "" {
		fmt.Fprintf(out, "  kind: %s\n", doc.Kind)
	}
	fmt.Fprintf(out, "  id: %s\n", doc.ID)
	fmt.Fprintf(out, "  title: %s\n", doc.Title)
	fmt.Fprintf(out, "  status: %s\n", doc.Status)
	fmt.Fprintf(out, "  date: %s\n", doc.Date)
	if len(doc.Tags) > 0 {
		fmt.Fprintf(out, "  tags: %s\n", strings.Join(doc.Tags, ", "))
	}
	if doc.SupersededBy != "" {
		fmt.Fprintf(out, "  superseded_by: %s\n", doc.SupersededBy)
	}
	if len(doc.Supersedes) > 0 {
		fmt.Fprintf(out, "  supersedes: %s\n", strings.Join(doc.Supersedes, ", "))
	}
	if doc.DeprecatedBy != "" {
		fmt.Fprintf(out, "  deprecated_by: %s\n", doc.DeprecatedBy)
	}
	fmt.Fprintf(out, "  path: %s\n", doc.Path)
	if includeContent && doc.Content != "" {
		fmt.Fprintln(out, "  content:")
		for _, line := range strings.Split(doc.Content, "\n") {
			fmt.Fprintf(out, "    %s\n", line)
		}
	}
}
