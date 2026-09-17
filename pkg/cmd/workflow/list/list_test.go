package list

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/workflow/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ListOptions
		wantsErr bool
	}{
		{
			name: "no arguments",
			wants: ListOptions{
				Limit: defaultLimit,
			},
		},
		{
			name: "all flag",
			cli:  "--all",
			wants: ListOptions{
				Limit: defaultLimit,
				All:   true,
			},
		},
		{
			name: "limit flag",
			cli:  "--limit 100",
			wants: ListOptions{
				Limit: 100,
			},
		},
		{
			name:     "invalid limit flag",
			cli:      "--limit 0",
			wantsErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()

			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
			assert.Equal(t, tt.wants.All, gotOpts.All)
		})
	}
}

func TestListRun(t *testing.T) {
	workflows := []shared.Workflow{
		{
			Name:  "Go",
			State: shared.Active,
			ID:    707,
		},
		{
			Name:  "Linter",
			State: shared.Active,
			ID:    666,
		},
		{
			Name:  "Release",
			State: shared.DisabledManually,
			ID:    451,
		},
	}
	payload := shared.WorkflowsPayload{Workflows: workflows}

	tests := []struct {
		name       string
		opts       *ListOptions
		wantErr    bool
		wantOut    string
		wantErrOut string
		stubs      func(*httpmock.Registry)
		tty        bool
	}{
		{
			name: "lists worrkflows nontty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			wantOut: "workflows[2]{id,name,state,path}:\n  707,Go,active,\n  666,Linter,active,\n\ncount: 2 of 2\n",
		},
		{
			name: "lists workflows tty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			tty:     true,
			wantOut: "NAME    STATE   ID\nGo      active  707\nLinter  active  666\n",
		},
		{
			name: "lists workflows with limit tty",
			opts: &ListOptions{
				Limit: 1,
			},
			tty:     true,
			wantOut: "NAME  STATE   ID\nGo    active  707\n",
		},
		{
			name: "show all workflows tty",
			opts: &ListOptions{
				Limit: defaultLimit,
				All:   true,
			},
			tty:     true,
			wantOut: "NAME     STATE              ID\nGo       active             707\nLinter   active             666\nRelease  disabled_manually  451\n",
		},
		{
			name: "no results nontty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{}),
				)
			},
			// Non-TTY empty results are a well-formed empty TOON state, not an error.
			wantOut: "workflows[0]{id,name,state,path}:\n\ncount: 0 of 0\n",
		},
		{
			name: "paginates workflows nontty",
			opts: &ListOptions{
				Limit: 101,
			},
			stubs: func(reg *httpmock.Registry) {
				workflows := []shared.Workflow{}
				var flowID int64
				for flowID = range 103 {
					workflows = append(workflows, shared.Workflow{
						ID:    flowID,
						Name:  fmt.Sprintf("flow %d", flowID),
						State: shared.Active,
					})
				}
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{
						Workflows: workflows[0:100],
					}))
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{
						Workflows: workflows[100:],
					}))
			},
			wantOut: longOutputTOON(101),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			defer reg.Verify(t)
			if tt.stubs == nil {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(payload),
				)
			} else {
				tt.stubs(reg)
			}

			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}

			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			tt.opts.IO = ios

			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.FromFullName("OWNER/REPO")
			}

			err := listRun(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantOut, stdout.String())
			assert.Equal(t, tt.wantErrOut, stderr.String())
		})
	}
}

// longOutputTOON builds the TOON output for the pagination test: 101 workflows.
func longOutputTOON(n int) string {
	out := fmt.Sprintf("workflows[%d]{id,name,state,path}:\n", n)
	for i := range n {
		out += fmt.Sprintf("  %d,flow %d,active,\n", i, i)
	}
	return out + fmt.Sprintf("\ncount: %d of %d\n", n, n)
}
