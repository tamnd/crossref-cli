package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	var workType string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Crossref works by full text",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching works for %q...", args[0])
			works, err := a.client.SearchWorks(cmd.Context(), args[0], n, workType)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(works, len(works))
		},
	}
	cmd.Flags().StringVar(&workType, "type", "", "filter by work type, e.g. journal-article")
	return cmd
}

func (a *App) typesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List all Crossref work type identifiers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.progressf("fetching work types...")
			types, err := a.client.ListTypes(cmd.Context())
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(types, len(types))
		},
	}
}
