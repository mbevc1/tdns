package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an initial config.json file in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		configFile := "config.json"

		// Define the structure of the data
		data := map[string]string{
			"token": "",
			"host":  "http://localhost:5380",
		}

		// Check if file exists
		if _, err := os.Stat(configFile); err == nil {
			fmt.Printf("⚠️  file %s already exists!\n", configFile)
			return
		}

		// Create the JSON file
		file, err := os.Create(configFile)
		if err != nil {
			fmt.Println("❌ Error creating file:", err)
			return
		}
		defer file.Close()

		// Encode the data as JSON and write it to the file
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ") // Pretty print
		if err := encoder.Encode(data); err != nil {
			fmt.Println("❌ Error encoding JSON:", err)
			return
		}

		absPath, _ := filepath.Abs(configFile)
		fmt.Printf("✅ Successfully created config file at %s\n", absPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
