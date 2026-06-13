package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) workCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "work <doi>",
		Short: "Fetch a single work by DOI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doi := args[0]
			a.progressf("fetching work %q...", doi)
			work, err := a.client.GetWork(cmd.Context(), doi)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]interface{}{work})
		},
	}
}
