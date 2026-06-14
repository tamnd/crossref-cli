package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) fundersCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "funders",
		Short: "Search research funders registered with Crossref",
		Long: `Search research funders in the Crossref funder registry.

Funders are organizations that award research grants. The Crossref Funder
Registry assigns each a stable identifier used to tag DOI metadata.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching funders for %q...", query)
			funders, err := a.client.SearchFunders(cmd.Context(), query, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(funders, len(funders))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "search query (name keyword)")
	return cmd
}

func (a *App) membersCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Search Crossref publisher members",
		Long: `Search Crossref publisher members by name.

Members are publishers and institutions that register DOIs with Crossref.
Each member has a numeric ID and one or more DOI prefix registrations.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching members for %q...", query)
			members, err := a.client.SearchMembers(cmd.Context(), query, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(members, len(members))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "search query (publisher name)")
	return cmd
}
