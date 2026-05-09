package cli

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "groups",
		Aliases: []string{"group"},
		Short:   "List and describe consumer groups",
	}
	cmd.AddCommand(
		newGroupsListCmd(),
		newGroupsDescribeCmd(),
	)
	return cmd
}

func newGroupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List consumer groups",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			gs, err := adm.ListGroups(ctx)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(gs))
			for n := range gs {
				names = append(names, n)
			}
			sort.Strings(names)

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "GROUP\tSTATE\tPROTOCOL")
			for _, n := range names {
				g := gs[n]
				fmt.Fprintf(tw, "%s\t%s\t%s\n", n, g.State, g.ProtocolType)
			}
			return tw.Flush()
		},
	}
}

func newGroupsDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe <group>",
		Short: "Describe a consumer group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			gs, err := adm.DescribeGroups(ctx, args[0])
			if err != nil {
				return err
			}
			g, ok := gs[args[0]]
			if !ok {
				return fmt.Errorf("group %q not found", args[0])
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Group:    %s\n", g.Group)
			fmt.Fprintf(out, "State:    %s\n", g.State)
			fmt.Fprintf(out, "Protocol: %s/%s\n", g.ProtocolType, g.Protocol)
			fmt.Fprintf(out, "Members:  %d\n\n", len(g.Members))

			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "MEMBER\tCLIENT-ID\tHOST")
			for _, m := range g.Members {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", m.MemberID, m.ClientID, m.ClientHost)
			}
			return tw.Flush()
		},
	}
}
