package cli

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newMetadataCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "metadata",
		Aliases: []string{"meta"},
		Short:   "Show cluster metadata (brokers, controller)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			md, err := adm.BrokerMetadata(ctx)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Cluster:    %s\n", md.Cluster)
			fmt.Fprintf(out, "Controller: %d\n\n", md.Controller)

			brokers := make([]int32, 0, len(md.Brokers))
			byID := make(map[int32]struct {
				Host string
				Port int32
				Rack *string
			}, len(md.Brokers))
			for _, b := range md.Brokers {
				brokers = append(brokers, b.NodeID)
				byID[b.NodeID] = struct {
					Host string
					Port int32
					Rack *string
				}{Host: b.Host, Port: b.Port, Rack: b.Rack}
			}
			sort.Slice(brokers, func(i, j int) bool { return brokers[i] < brokers[j] })

			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "BROKER\tHOST\tPORT\tRACK")
			for _, id := range brokers {
				b := byID[id]
				rack := ""
				if b.Rack != nil {
					rack = *b.Rack
				}
				fmt.Fprintf(tw, "%d\t%s\t%d\t%s\n", id, b.Host, b.Port, rack)
			}
			return tw.Flush()
		},
	}
}
