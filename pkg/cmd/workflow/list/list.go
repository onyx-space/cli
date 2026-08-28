package list

import (
	"fmt"
	"net/http"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/internal/toon"
	"github.com/cli/cli/v2/pkg/cmd/workflow/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

const defaultLimit = 50

type ListOptions struct {
	IO         *iostreams.IOStreams
	HttpClient func() (*http.Client, error)
	BaseRepo   func() (ghrepo.Interface, error)
	Exporter   cmdutil.Exporter

	All   bool
	Limit int
}

var workflowFields = []string{
	"id",
	"name",
	"path",
	"state",
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List workflows",
		Long:    "List workflow files, hiding disabled workflows by default.",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// support `-R, --repo` override
			opts.BaseRepo = f.BaseRepo

			if opts.Limit < 1 {
				return cmdutil.FlagErrorf("invalid limit: %v", opts.Limit)
			}

			if runF != nil {
				return runF(opts)
			}

			return listRun(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultLimit, "Maximum number of workflows to fetch")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Include disabled workflows")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, workflowFields)
	return cmd
}

func listRun(opts *ListOptions) error {
	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	httpClient, err := opts.HttpClient()
	if err != nil {
		return fmt.Errorf("could not create http client: %w", err)
	}
	client := api.NewClientFromHTTP(httpClient)

	opts.IO.StartProgressIndicator()
	workflows, err := shared.GetWorkflows(client, repo, opts.Limit)
	opts.IO.StopProgressIndicator()
	if err != nil {
		return fmt.Errorf("could not get workflows: %w", err)
	}

	var filteredWorkflows []shared.Workflow
	if opts.All {
		filteredWorkflows = workflows
	} else {
		for _, workflow := range workflows {
			if !workflow.Disabled() {
				filteredWorkflows = append(filteredWorkflows, workflow)
			}
		}
	}

	if len(filteredWorkflows) == 0 {
		if !opts.IO.IsStdoutTTY() {
			// R4 empty-state contract: well-formed empty TOON, exit 0.
			fmt.Fprintf(opts.IO.Out, "workflows[0]{id,name,state,path}:\n\ncount: 0 of 0\n")
			return nil
		}
		return cmdutil.NewNoResultsError("no workflows found")
	}

	if err := opts.IO.StartPager(); err == nil {
		defer opts.IO.StopPager()
	} else {
		fmt.Fprintf(opts.IO.ErrOut, "failed to start pager: %v\n", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, filteredWorkflows)
	}

	if !opts.IO.IsStdoutTTY() {
		// TOON array output — AXI contract (mirrors gh-axi's `workflow list` schema).
		fmt.Fprintf(opts.IO.Out, "workflows[%d]{id,name,state,path}:\n", len(filteredWorkflows))
		for _, w := range filteredWorkflows {
			fmt.Fprintf(opts.IO.Out, "  %d,%s,%s,%s\n",
				w.ID, toon.Quote(w.Name), string(w.State), toon.Quote(w.Path))
		}
		fmt.Fprintf(opts.IO.Out, "\ncount: %d of %d\n", len(filteredWorkflows), len(filteredWorkflows))
		return nil
	}

	cs := opts.IO.ColorScheme()
	tp := tableprinter.New(opts.IO, tableprinter.WithHeader("Name", "State", "ID"))

	for _, workflow := range filteredWorkflows {
		tp.AddField(workflow.Name)
		tp.AddField(string(workflow.State))
		tp.AddField(fmt.Sprintf("%d", workflow.ID), tableprinter.WithColor(cs.Cyan))
		tp.EndRow()
	}

	return tp.Render()
}
