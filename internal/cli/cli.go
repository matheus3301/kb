// Package cli implements the kb command tree without process-global state.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// Dependencies keeps filesystem construction out of command parsing and tests.
type Dependencies struct {
	ResolveRoot func(string) (string, error)
	Open        func(string) (mcpserver.Reader, mcpserver.Writer, error)
	Init        func(context.Context, string, bool) (mcpserver.Reader, mcpserver.Writer, error)
	Version     string
	Commit      string
}

type reported struct{ code int }

func (e reported) Error() string { return "command failed" }

type runner struct {
	deps                 Dependencies
	in                   io.Reader
	out, errOut          io.Writer
	root, asOf           string
	jsonOutput, readOnly bool
}

// Run emits one result envelope for data commands, and reserves stderr for errors.
func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer, deps Dependencies) int {
	r := &runner{deps: deps, in: in, out: out, errOut: errOut}
	c := r.command()
	c.SetArgs(args)
	c.SetIn(in)
	c.SetOut(out)
	c.SetErr(errOut)
	selected, err := c.ExecuteContextC(ctx)
	if err == nil {
		return 0
	}
	var done reported
	if errors.As(err, &done) {
		return done.code
	}
	if selected != nil && selected.Name() == "mcp" {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}
	return r.emit(mcpserver.Failure(r.root, mcpserver.Invalid(err.Error())))
}

func (r *runner) emit(e mcpserver.Envelope) int {
	if err := json.NewEncoder(r.out).Encode(e); err != nil {
		_, _ = fmt.Fprintln(r.errOut, "unable to write response")
		return 6
	}
	if !e.OK {
		_, _ = fmt.Fprintln(r.errOut, e.Error.Error())
	}
	return mcpserver.ExitCode(e)
}

func (r *runner) result(e mcpserver.Envelope) error {
	if code := r.emit(e); code != 0 {
		return reported{code}
	}
	return nil
}

func (r *runner) open(ctx context.Context, init, para bool) (*mcpserver.Adapter, error) {
	var asOf time.Time
	if r.asOf != "" {
		var err error
		asOf, err = time.Parse(time.RFC3339, r.asOf)
		if err != nil {
			return nil, mcpserver.Invalid("--as-of must be an RFC3339 timestamp with timezone")
		}
	}
	root := r.root
	if init {
		if root == "" {
			root = os.Getenv("KB_ROOT")
		}
		if root == "" {
			root = "."
		}
	} else {
		var err error
		root, err = r.deps.ResolveRoot(root)
		if err != nil {
			return nil, err
		}
	}
	var reader mcpserver.Reader
	var writer mcpserver.Writer
	var err error
	if init {
		if r.readOnly {
			return nil, &api.DomainError{Code: "READ_ONLY", Message: "writes are disabled"}
		}
		reader, writer, err = r.deps.Init(ctx, root, para)
	} else {
		reader, writer, err = r.deps.Open(root)
	}
	if err != nil {
		return nil, err
	}
	if !asOf.IsZero() {
		if clock, ok := reader.(interface{ SetAsOf(time.Time) }); ok {
			clock.SetAsOf(asOf)
		}
	}
	return &mcpserver.Adapter{Reader: reader, Writer: writer, ReadOnly: r.readOnly}, nil
}

func (r *runner) command() *cobra.Command {
	c := &cobra.Command{Use: "kb", Short: "Local Markdown Knowledge Base for humans and AI agents", SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	c.CompletionOptions.DisableDefaultCmd = true
	c.PersistentFlags().StringVar(&r.root, "root", "", "Knowledge Base root (then KB_ROOT, then ancestor discovery)")
	c.PersistentFlags().BoolVar(&r.jsonOutput, "json", false, "Emit a kb.v1 JSON envelope (also the default)")
	c.PersistentFlags().BoolVar(&r.readOnly, "read-only", false, "Disable writes and hide MCP write tools")
	c.PersistentFlags().StringVar(&r.asOf, "as-of", "", "Evaluate staleness at an RFC3339 timestamp")
	var para bool
	initCmd := &cobra.Command{Use: "init", Short: "Initialize missing base files", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		a, err := r.open(cmd.Context(), true, para)
		if err != nil {
			return r.result(mcpserver.Failure(r.root, err))
		}
		return r.result(a.Call(cmd.Context(), "status", nil))
	}}
	initCmd.Flags().BoolVar(&para, "para", false, "Create initial PARA navigation")
	c.AddCommand(initCmd)
	for _, action := range []string{"status", "list", "search", "read", "context", "links", "create", "patch", "move", "archive", "apply", "validate", "maintain", "recover", "export", "backup", "restore"} {
		c.AddCommand(r.dataCommand(action))
	}
	index := &cobra.Command{Use: "index", Short: "Manage the disposable search index", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	rebuild := r.dataCommand("rebuild")
	index.AddCommand(rebuild)
	c.AddCommand(index)
	c.AddCommand(&cobra.Command{Use: "version", Short: "Show build version", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error {
		return r.result(mcpserver.Envelope{SchemaVersion: "kb.v1", OK: true, Data: map[string]string{"version": r.deps.Version, "commit": r.deps.Commit}, Warnings: []api.Diagnostic{}, Meta: mcpserver.Meta{}})
	}})
	c.AddCommand(&cobra.Command{Use: "mcp", Short: "Serve the fixed Knowledge Base over stdio MCP", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		a, err := r.open(cmd.Context(), false, false)
		if err != nil {
			_, _ = fmt.Fprintln(r.errOut, err)
			return reported{mcpserver.ExitCode(mcpserver.Failure(r.root, err))}
		}
		reader, ok := r.in.(io.ReadCloser)
		if !ok {
			reader = io.NopCloser(r.in)
		}
		err = mcpserver.New(a, r.deps.Version).Run(cmd.Context(), &mcp.IOTransport{Reader: reader, Writer: writeCloser{r.out}, MaxLineLength: api.MaxPlanBytes + (64 << 10)})
		if err != nil && !errors.Is(err, context.Canceled) {
			_, _ = fmt.Fprintln(r.errOut, err)
			return reported{6}
		}
		return nil
	}})
	return c
}

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error { return nil }

func (r *runner) input(ctx context.Context, file string) (map[string]any, error) {
	if file == "" {
		return map[string]any{}, nil
	}
	var data []byte
	var err error
	if file == "-" {
		if closer, ok := r.in.(io.Closer); ok {
			stop := context.AfterFunc(ctx, func() { _ = closer.Close() })
			defer stop()
		}
		data, err = mcpserver.ReadInput(r.in)
	} else {
		var f *os.File
		f, err = os.Open(file) // #nosec G304 -- The user explicitly selects this bounded JSON request file.
		if err != nil {
			return nil, err
		}
		data, err = mcpserver.ReadInput(f)
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err = mcpserver.Decode(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *runner) dataCommand(action string) *cobra.Command {
	use := action
	maxArgs := 0
	switch action {
	case "search", "context":
		use += " [QUERY]"
		maxArgs = 1
	case "read", "links", "patch", "archive":
		use += " [PATH]"
		maxArgs = 1
	case "move":
		use += " [PATH] [TO]"
		maxArgs = 2
	case "export":
		use += " [DESTINATION]"
		maxArgs = 1
	case "backup":
		use += " [ARCHIVE]"
		maxArgs = 1
	case "restore":
		use += " [ARCHIVE] [DESTINATION]"
		maxArgs = 2
	}
	descriptions := map[string]string{
		"rebuild":  "Rebuild the disposable search index",
		"status":   "Inspect the Knowledge Base scope, generation, and diagnostics",
		"list":     "List notes with filters and cursor pagination",
		"search":   "Search notes with lexical ranking and citable snippets",
		"read":     "Read a note, section, or physical line range",
		"context":  "Collect bounded source excerpts with citations",
		"links":    "Inspect incoming and outgoing note references",
		"create":   "Create a typed OKF note with an idempotent request",
		"patch":    "Edit a note using its current revision",
		"move":     "Relocate a note and update resolved references",
		"archive":  "Move a note into the configured Archive directory",
		"apply":    "Preview or apply a coordinated revision-checked plan",
		"validate": "Check OKF conformance or structural health",
		"maintain": "Preview or apply managed index maintenance",
		"recover":  "Inspect or recover an interrupted transaction",
		"export":   "Export a validated snapshot with a scope manifest",
		"backup":   "Back up content, configuration and durable receipts to .kb/backups",
		"restore":  "Restore an operational archive into a new .kb/restores directory",
	}
	cmd := &cobra.Command{Use: use, Short: descriptions[action], Args: cobra.MaximumNArgs(maxArgs)}
	var input string
	var dry, apply, repair bool
	cmd.Flags().StringVar(&input, "input", "", "JSON request file, or - for stdin (maximum 4 MiB)")
	if mcpserver.IsWrite(action) && action != "backup" && action != "restore" {
		cmd.Flags().BoolVar(&dry, "dry-run", false, "Preview without publishing changes")
	}
	if action == "apply" {
		cmd.Flags().BoolVar(&repair, "repair", false, "Allow an explicit repair plan")
	}
	if action == "maintain" {
		cmd.Flags().BoolVar(&apply, "apply", false, "Apply structural maintenance")
	}
	stringsFlags := map[string]string{}
	intFlags := map[string]int{}
	boolFlags := map[string]bool{}
	filterLists := map[string][]string{}
	switch action {
	case "list", "search", "context":
		cmd.Flags().String("prefix", "", "Scoped path prefix")
		for _, name := range []string{"para", "type", "tags", "status", "trust"} {
			cmd.Flags().StringSlice(name, nil, "Filter notes by "+name)
			filterLists[name] = nil
		}
		for _, name := range []string{"stale", "has-sources", "include-archives", "include-deprecated"} {
			cmd.Flags().Bool(name, false, "Filter option: "+name)
		}
		if action == "context" {
			intFlags["max-tokens"] = api.DefaultContextTokens
			intFlags["max-bytes"] = api.MaxReadBytes
			boolFlags["include-links"] = false
		} else {
			intFlags["limit"] = api.DefaultHits
			stringsFlags["cursor"] = ""
			boolFlags["literal"] = false
		}
	case "read":
		stringsFlags["section"] = ""
		intFlags["line-start"] = 0
		intFlags["line-end"] = 0
		boolFlags["raw"] = false
		boolFlags["include-links"] = false
	case "links":
		stringsFlags["direction"] = "both"
	case "validate":
		stringsFlags["profile"] = "okf"
		boolFlags["strict"] = false
	case "apply":
		boolFlags["incremental"] = false
	}
	for name, value := range stringsFlags {
		cmd.Flags().String(name, value, "Request "+name)
	}
	for name, value := range intFlags {
		cmd.Flags().Int(name, value, "Request "+name)
	}
	for name, value := range boolFlags {
		cmd.Flags().Bool(name, value, "Request "+name)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		a, err := r.open(cmd.Context(), false, false)
		if err != nil {
			return r.result(mcpserver.Failure(r.root, err))
		}
		requestRoot := a.Reader.Root()
		if a.ReadOnly && mcpserver.IsWrite(action) {
			return r.result(a.Call(cmd.Context(), action, nil))
		}
		values, err := r.input(cmd.Context(), input)
		if err != nil {
			return r.result(mcpserver.Failure(requestRoot, err))
		}
		if (action == "create" || action == "patch" || action == "move" || action == "archive" || action == "apply") && input == "" {
			return r.result(mcpserver.Failure(requestRoot, mcpserver.Invalid("--input is required for writes")))
		}
		for i, arg := range args {
			key := "path"
			if action == "search" || action == "context" {
				key = "query"
			}
			if action == "export" {
				key = "destination"
			}
			if action == "backup" || action == "restore" {
				key = "archive"
				if i == 1 {
					key = "destination"
				}
			}
			if i == 1 && action != "restore" {
				key = "to"
			}
			if old, ok := values[key]; ok && old != arg {
				return r.result(mcpserver.Failure(requestRoot, mcpserver.Invalid("positional argument conflicts with input "+key)))
			}
			values[key] = arg
		}
		for name := range stringsFlags {
			if cmd.Flags().Changed(name) {
				v, _ := cmd.Flags().GetString(name)
				values[jsonKey(name)] = v
			}
		}
		for name := range intFlags {
			if cmd.Flags().Changed(name) {
				v, _ := cmd.Flags().GetInt(name)
				values[jsonKey(name)] = v
			}
		}
		for name := range boolFlags {
			if cmd.Flags().Changed(name) {
				v, _ := cmd.Flags().GetBool(name)
				values[jsonKey(name)] = v
			}
		}
		if action == "list" || action == "search" || action == "context" {
			filters, ok := values["filters"].(map[string]any)
			if !ok {
				if _, exists := values["filters"]; exists {
					return r.result(mcpserver.Failure(requestRoot, mcpserver.Invalid("filters must be an object")))
				}
				filters = map[string]any{}
			}
			for name := range filterLists {
				if cmd.Flags().Changed(name) {
					v, _ := cmd.Flags().GetStringSlice(name)
					filters[name] = v
				}
			}
			if cmd.Flags().Changed("prefix") {
				v, _ := cmd.Flags().GetString("prefix")
				filters["prefix"] = v
			}
			for _, name := range []string{"stale", "has-sources", "include-archives", "include-deprecated"} {
				if cmd.Flags().Changed(name) {
					v, _ := cmd.Flags().GetBool(name)
					filters[jsonKey(name)] = v
				}
			}
			if len(filters) > 0 {
				values["filters"] = filters
			}
		}
		if mcpserver.IsWrite(action) && cmd.Flags().Changed("dry-run") {
			values["dry_run"] = dry
		}
		if action == "maintain" {
			values["dry_run"] = !apply || dry
		}
		if action == "apply" && cmd.Flags().Changed("repair") {
			values["repair"] = repair
		}
		raw, err := json.Marshal(values)
		if err != nil {
			return r.result(mcpserver.Failure(requestRoot, err))
		}
		return r.result(a.Call(cmd.Context(), action, raw))
	}
	return cmd
}

func jsonKey(flag string) string {
	b := []byte(flag)
	for i := range b {
		if b[i] == '-' {
			b[i] = '_'
		}
	}
	return string(b)
}
