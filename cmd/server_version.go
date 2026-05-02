package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var serverVersionCmd = &cobra.Command{
	Use:     "server-version",
	Aliases: []string{"sv"},
	Short:   "Print the version of the connected DNS server",
	Run: func(cmd *cobra.Command, args []string) {
		v, err := api.New().ServerVersion()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Println(v)
	},
}

func init() {
	rootCmd.AddCommand(serverVersionCmd)
}
