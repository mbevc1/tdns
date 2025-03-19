package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a config.json file from config-example.json",
	Run: func(cmd *cobra.Command, args []string) {
		src := "config-example.json"
		dst := "config.json"

		if _, err := os.Stat(dst); err == nil {
			fmt.Println("⚠️  config.json already exists.")
			return
		}

		srcFile, err := os.Open(src)
		if err != nil {
			fmt.Printf("❌ Failed to read %s: %v\n", src, err)
			return
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dst)
		if err != nil {
			fmt.Printf("❌ Failed to create %s: %v\n", dst, err)
			return
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			fmt.Printf("❌ Failed to copy: %v\n", err)
			return
		}

		absPath, _ := filepath.Abs(dst)
		fmt.Printf("✅ Created config file at %s\n", absPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
