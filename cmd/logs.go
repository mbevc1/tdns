package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		token := viper.GetString("token")
		host := viper.GetString("host")

		url := fmt.Sprintf("%s/api/logs/list?token=%s", host, token)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Invalid response: %v\n", err)
			os.Exit(1)
		}

		if status, ok := result["status"].(string); !ok || status != "ok" {
			if msg, ok := result["errorMessage"].(string); ok {
				fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
			} else {
				fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
			}
			os.Exit(1)
		}

		logs := result["response"].(map[string]interface{})["logFiles"].([]interface{})
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
		token := viper.GetString("token")
		host := viper.GetString("host")
		fileName := args[0]

		url := fmt.Sprintf("%s/api/logs/download?token=%s&fileName=%s", host, token, fileName)
		resp, err := http.Get(url)
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
		token := viper.GetString("token")
		host := viper.GetString("host")
		fileName := args[0]

		url := fmt.Sprintf("%s/api/logs/delete?token=%s&log=%s", host, token, fileName)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Invalid response: %v\n", err)
			os.Exit(1)
		}

		if status, ok := result["status"].(string); !ok || status != "ok" {
			if msg, ok := result["errorMessage"].(string); ok {
				fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
			} else {
				fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
			}
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
		token := viper.GetString("token")
		host := viper.GetString("host")

		fmt.Print("Are you sure you want to delete ALL logs? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("❌ Aborted.")
			return
		}

		url := fmt.Sprintf("%s/api/logs/deleteAll?token=%s", host, token)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Invalid response: %v\n", err)
			os.Exit(1)
		}

		if status, ok := result["status"].(string); !ok || status != "ok" {
			if msg, ok := result["errorMessage"].(string); ok {
				fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
			} else {
				fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
			}
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
