package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(),
				"kafko %s\ncommit:  %s\nbuilt:   %s\ngo:      %s\nos/arch: %s/%s\n",
				Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
			return err
		},
	}
}
