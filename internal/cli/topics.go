package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newTopicsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "topics",
		Aliases: []string{"topic"},
		Short:   "List, describe, create, and delete topics",
	}
	cmd.AddCommand(
		newTopicsListCmd(),
		newTopicsDescribeCmd(),
		newTopicsCreateCmd(),
		newTopicsDeleteCmd(),
	)
	return cmd
}

func newTopicsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List topics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			details, err := adm.ListTopics(ctx)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(details))
			for n := range details {
				names = append(names, n)
			}
			sort.Strings(names)

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "TOPIC\tPARTITIONS\tREPLICATION")
			for _, n := range names {
				t := details[n]
				rf := 0
				for _, p := range t.Partitions {
					rf = len(p.Replicas)
					break
				}
				fmt.Fprintf(tw, "%s\t%d\t%d\n", n, len(t.Partitions), rf)
			}
			return tw.Flush()
		},
	}
}

func newTopicsDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe <topic>",
		Short: "Describe a topic (partitions, leader, ISR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			details, err := adm.ListTopics(ctx, args[0])
			if err != nil {
				return err
			}
			t, ok := details[args[0]]
			if !ok {
				return fmt.Errorf("topic %q not found", args[0])
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Topic:      %s\n", t.Topic)
			fmt.Fprintf(out, "Partitions: %d\n\n", len(t.Partitions))

			parts := make([]int32, 0, len(t.Partitions))
			for p := range t.Partitions {
				parts = append(parts, p)
			}
			sort.Slice(parts, func(i, j int) bool { return parts[i] < parts[j] })

			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "PARTITION\tLEADER\tREPLICAS\tISR")
			for _, p := range parts {
				pt := t.Partitions[p]
				fmt.Fprintf(tw, "%d\t%d\t%v\t%v\n", p, pt.Leader, pt.Replicas, pt.ISR)
			}
			return tw.Flush()
		},
	}
}

func newTopicsCreateCmd() *cobra.Command {
	var partitions int32
	var replication int16
	cmd := &cobra.Command{
		Use:   "create <topic>",
		Short: "Create a topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			res, err := adm.CreateTopic(ctx, partitions, replication, nil, args[0])
			if err != nil {
				return err
			}
			if res.Err != nil {
				return res.Err
			}
			fmt.Fprintf(cmd.OutOrStdout(),
				"topic %q created (partitions=%d, replication=%d)\n",
				args[0], partitions, replication)
			return nil
		},
	}
	cmd.Flags().Int32VarP(&partitions, "partitions", "p", 1, "number of partitions")
	cmd.Flags().Int16VarP(&replication, "replication", "r", 1, "replication factor")
	return cmd
}

func newTopicsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <topic>...",
		Aliases: []string{"rm"},
		Short:   "Delete one or more topics",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := adminContext()
			defer cancel()

			adm, closeFn, err := newAdminClient()
			if err != nil {
				return err
			}
			defer closeFn()

			res, err := adm.DeleteTopics(ctx, args...)
			if err != nil {
				return err
			}
			var hadErr bool
			for name, r := range res {
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "kafko: failed to delete %q: %v\n", name, r.Err)
					hadErr = true
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "topic %q deleted\n", name)
			}
			if hadErr {
				return fmt.Errorf("one or more deletes failed")
			}
			return nil
		},
	}
}
