package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/perlinson/codex-session-restore/internal/codex"
)

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		writeUsage(stdout)
		return nil
	}

	switch args[0] {
	case "inspect":
		return runInspect(args[1:], stdout)
	case "search":
		return runSearch(args[1:], stdout)
	case "repair":
		return runRepair(args[1:], stdout)
	case "switch-current-provider", "switch-provider":
		return runSwitchCurrentProvider(args[1:], stdout)
	case "-h", "--help", "help":
		writeUsage(stdout)
		return nil
	default:
		writeUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, `codex-session-restore

Commands:
  inspect                  List recent threads from state_5.sqlite
  search                   Search rollout JSONL files for keywords
  repair                   Back up and repair selected missing-session metadata
  switch-current-provider  Move sessions from other providers to the current provider

Examples:
  codex-session-restore inspect --limit 15
  codex-session-restore search --keyword godot --keyword project
  codex-session-restore repair --thread-id THREAD_ID --reference-thread-id VISIBLE_THREAD_ID
  codex-session-restore repair --thread-id THREAD_ID --model-provider custom --cwd "/Users/me/project"
  codex-session-restore switch-current-provider --dry-run
  codex-session-restore switch-current-provider`)
}

func runInspect(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	codexDir := fs.String("codex-dir", codex.DefaultCodexDir(), "Path to the .codex directory")
	limit := fs.Int("limit", 20, "Number of threads to show")
	asJSON := fs.Bool("json", false, "Print results as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workspace, err := codex.OpenWorkspace(*codexDir, false)
	if err != nil {
		return err
	}
	threads, err := workspace.RecentThreads(*limit)
	if err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(stdout, threads)
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPROVIDER\tSOURCE\tCLI_VERSION\tARCHIVED\tUPDATED\tTITLE\tCWD")
	for _, thread := range threads {
		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			thread.ID,
			thread.ModelProvider,
			thread.Source,
			thread.CLIVersion,
			thread.Archived,
			formatUnix(thread.UpdatedAt),
			thread.Title,
			thread.CWD,
		)
	}
	return tw.Flush()
}

func runSearch(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	codexDir := fs.String("codex-dir", codex.DefaultCodexDir(), "Path to the .codex directory")
	limit := fs.Int("limit", 20, "Maximum matches to print")
	var keywords stringList
	fs.Var(&keywords, "keyword", "Keyword to match; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}

	keywords = append(keywords, fs.Args()...)
	if len(keywords) == 0 {
		return errors.New("provide at least one keyword via --keyword or positional args")
	}

	workspace, err := codex.OpenWorkspace(*codexDir, false)
	if err != nil {
		return err
	}
	results, err := workspace.SearchTranscripts(keywords, *limit)
	if err != nil {
		return err
	}
	for _, result := range results {
		fmt.Fprintf(stdout, "%s:%d %s\n", result.Path, result.LineNumber, result.Line)
	}
	return nil
}

func runRepair(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("repair", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	codexDir := fs.String("codex-dir", codex.DefaultCodexDir(), "Path to the .codex directory")
	referenceID := fs.String("reference-thread-id", "", "Visible thread id to copy provider/cwd from")
	modelProvider := fs.String("model-provider", "", "Override model_provider")
	cwd := fs.String("cwd", "", "Override cwd")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	var threadIDs stringList
	fs.Var(&threadIDs, "thread-id", "Thread id to repair; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}

	threadIDs = append(threadIDs, fs.Args()...)
	if len(threadIDs) == 0 {
		return errors.New("provide at least one thread id via --thread-id or positional args")
	}

	workspace, err := codex.OpenWorkspace(*codexDir, *dryRun)
	if err != nil {
		return err
	}
	report, err := workspace.RepairThreads(codex.RepairOptions{
		ThreadIDs:         threadIDs,
		ReferenceThreadID: strings.TrimSpace(*referenceID),
		ModelProvider:     strings.TrimSpace(*modelProvider),
		CWD:               strings.TrimSpace(*cwd),
		DryRun:            *dryRun,
	})
	if err != nil {
		return err
	}

	printRepairReport(stdout, report)
	return nil
}

func runSwitchCurrentProvider(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("switch-current-provider", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	codexDir := fs.String("codex-dir", codex.DefaultCodexDir(), "Path to the .codex directory")
	includeArchived := fs.Bool("include-archived", false, "Also switch archived sessions and unarchive them")
	limit := fs.Int("limit", 0, "Maximum sessions to switch; 0 means all")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workspace, err := codex.OpenWorkspace(*codexDir, *dryRun)
	if err != nil {
		return err
	}
	report, err := workspace.SwitchToCurrentProvider(codex.SwitchProviderOptions{
		IncludeArchived: *includeArchived,
		Limit:           *limit,
		DryRun:          *dryRun,
	})
	if err != nil {
		return err
	}

	printRepairReport(stdout, report)
	return nil
}

func printRepairReport(stdout io.Writer, report *codex.RepairReport) {
	fmt.Fprintf(stdout, "Database: %s\n", report.DatabasePath)
	if report.TargetProvider != "" {
		fmt.Fprintf(stdout, "Target provider: %s\n", report.TargetProvider)
	}
	fmt.Fprintf(stdout, "Session index: %s\n", report.SessionIndexPath)
	fmt.Fprintf(stdout, "Dry run: %t\n", report.DryRun)
	if len(report.BackupPaths) > 0 {
		fmt.Fprintln(stdout, "Backups:")
		for _, backup := range report.BackupPaths {
			fmt.Fprintf(stdout, "  %s\n", backup)
		}
	}
	fmt.Fprintln(stdout, "Repaired threads:")
	for _, repaired := range report.Threads {
		fmt.Fprintf(
			stdout,
			"  %s | provider=%s | cwd=%s | rollout_changed=%t | title=%s\n",
			repaired.ID,
			repaired.UpdatedProvider,
			repaired.UpdatedCWD,
			repaired.RolloutChanged,
			repaired.Title,
		)
	}
	if len(report.SkippedThreads) > 0 {
		fmt.Fprintln(stdout, "Skipped threads:")
		for _, skipped := range report.SkippedThreads {
			fmt.Fprintf(
				stdout,
				"  %s | rollout=%s | reason=%s | title=%s\n",
				skipped.ID,
				skipped.RolloutPath,
				skipped.Reason,
				skipped.Title,
			)
		}
	}
	fmt.Fprintf(stdout, "Session index updated: %t\n", report.SessionIndexUpdated)
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*s = append(*s, value)
	return nil
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func formatUnix(unix int64) string {
	if unix <= 0 {
		return "-"
	}
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}
