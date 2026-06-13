package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) journalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "journal <issn>",
		Short: "Fetch a journal by ISSN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issn := args[0]
			a.progressf("fetching journal %q...", issn)
			journal, err := a.client.GetJournal(cmd.Context(), issn)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]interface{}{journal})
		},
	}
}

func (a *App) journalsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "journals <query>",
		Short: "Search journals by title keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching journals for %q...", args[0])
			journals, err := a.client.SearchJournals(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(journals, len(journals))
		},
	}
}
