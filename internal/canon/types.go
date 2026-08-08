package canon

const (
	SchemaVersion = "canon.v1"

	defaultADRDir    = "docs/adr"
	defaultSpecDir   = "docs/spec"
	defaultDomainDir = "docs/domain"

	KindADR    = "adr"
	KindSPEC   = "spec"
	KindDomain = "domain"

	// PrefixADR, PrefixSPEC, and PrefixDomain are the stable id prefixes used
	// in filenames and selectors.
	PrefixADR    = "ADR-"
	PrefixSPEC   = "SPEC-"
	PrefixDomain = "DM-"
)

// Version is the canon build version. It defaults to "dev" and can be
// overridden at build time with
// -ldflags "-X github.com/victorhsb/canon/internal/canon.Version=v1.2.3".
var Version = "dev"

type Envelope struct {
	SchemaVersion string       `json:"schema_version"`
	Command       string       `json:"command"`
	Status        string       `json:"status"`
	Data          any          `json:"data,omitempty"`
	Warnings      []string     `json:"warnings,omitempty"`
	NextActions   []NextAction `json:"next_actions,omitempty"`
	Error         *CLIError    `json:"error,omitempty"`
}

type CLIError struct {
	Code         string       `json:"code"`
	Category     string       `json:"category"`
	Message      string       `json:"message"`
	SuggestedFix string       `json:"suggested_fix"`
	Diagnostics  []Diagnostic `json:"diagnostics,omitempty"`
}

type Diagnostic struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
	Path         string `json:"path,omitempty"`
	ID           string `json:"id,omitempty"`
}

type NextAction struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Safety      string `json:"safety"`
}

type ADR struct {
	Kind         string   `json:"kind"`
	ID           string   `json:"id"`
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Date         string   `json:"date"`
	Tags         []string `json:"tags,omitempty"`
	Supersedes   []string `json:"supersedes,omitempty"`
	SupersededBy string   `json:"superseded_by,omitempty"`
	DeprecatedBy string   `json:"deprecated_by,omitempty"`
	Path         string   `json:"path"`
	Content      string   `json:"content,omitempty"`
}

type Plan struct {
	DryRun      bool     `json:"dry_run"`
	ChangesMade bool     `json:"changes_made"`
	Operations  []OpPlan `json:"operations"`
}

type OpPlan struct {
	Action      string `json:"action"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type GlobalOptions struct {
	ADRDir    string
	SpecDir   string
	DomainDir string
	Format    string
}

// Document is an alias for ADR. The CLI manages multiple kinds of documents
// (ADR, SPEC, and domain entries today); they share the same parseable shape,
// so ADR remains the backing struct while Document is the preferred name in
// new code.
type Document = ADR
