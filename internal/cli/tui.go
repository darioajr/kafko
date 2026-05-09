package cli

import (
	"github.com/darioajr/kafko/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive Kafka TUI",
		Long: `Browse topics and tail messages interactively.

Keys (topic list): enter to tail, / to filter, q to quit
Keys (tail view):  esc to go back, / to filter, c to clear, q to quit`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := resolveClientOptions()
			if err != nil {
				return err
			}
			return tui.Run(opts)
		},
	}
}
