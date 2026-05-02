package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var outputPath string

var logsCmd = &cobra.Command{
	Use:     "logs",
	Aliases: []string{"lo"},
	Short:   "Interact with logs from the DNS system",
}

var logsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available log files",
	Run: func(cmd *cobra.Command, args []string) {
		_, response, err := api.New().GetJSON("/api/logs/list", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		logs := response["logFiles"].([]interface{})
		if len(logs) == 0 {
			fmt.Println("No log files found.")
			return
		}

		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Println(bold("Available Log Files:"))
		for _, entry := range logs {
			log := entry.(map[string]interface{})
			fmt.Printf("- %s (%s)\n", cyan(log["fileName"]), log["size"])
		}
	},
}

var logsDownloadCmd = &cobra.Command{
	Use:     "download [fileName]",
	Aliases: []string{"dl"},
	Short:   "Download a specific log file",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		resp, err := api.New().Get("/api/logs/download", url.Values{"fileName": {fileName}})
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("❌ Failed to download file: HTTP %d\n", resp.StatusCode)
			os.Exit(1)
		}

		outputFile := fileName + ".log"
		if outputPath != "" {
			outputFile = outputPath
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			fmt.Printf("Failed to save file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Log file saved as %s\n", outputFile)
	},
}

var logsDeleteCmd = &cobra.Command{
	Use:     "delete [fileName]",
	Aliases: []string{"de", "rm"},
	Short:   "Delete a specific log file",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		if _, _, err := api.New().GetJSON("/api/logs/delete", url.Values{"log": {fileName}}); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Log '%s' deleted successfully.\n", fileName)
	},
}

var logsDeleteAllCmd = &cobra.Command{
	Use:     "deleteAll",
	Aliases: []string{"da"},
	Short:   "Delete all log files",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Are you sure you want to delete ALL logs? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("❌ Aborted.")
			return
		}

		if _, _, err := api.New().GetJSON("/api/logs/deleteAll", nil); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ All logs deleted successfully.")
	},
}

func init() {
	logsDownloadCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Optional path to save the downloaded log file")
	logsCmd.AddCommand(logsListCmd)
	logsCmd.AddCommand(logsDownloadCmd)
	logsCmd.AddCommand(logsDeleteCmd)
	logsCmd.AddCommand(logsDeleteAllCmd)
	rootCmd.AddCommand(logsCmd)
}
