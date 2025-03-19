package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var getJSON bool

var backupOutputPath string

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage server settings",
}

var settingsBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Download a backup zip file of selected server settings",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		url := fmt.Sprintf("%s/api/settings/backup?token=%s&blockLists=true&logs=true&scopes=true&stats=true&zones=true&allowedZones=true&blockedZones=true&dnsSettings=true&logSettings=true&authConfig=true", host, token)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Failed to download backup: HTTP %d\n", resp.StatusCode)
			os.Exit(1)
		}

		outFile := backupOutputPath
		if outFile == "" {
			timestamp := time.Now().Format("20060102-150405")
			outFile = fmt.Sprintf("tdns-backup-%s.zip", timestamp)
		}

		outPath := filepath.Clean(outFile)
		out, err := os.Create(outPath)
		if err != nil {
			fmt.Printf("❌ Could not create file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			fmt.Printf("❌ Failed to write file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Backup saved as %s\n", outPath)
	},
}

func init() {
	settingsBackupCmd.Flags().StringVarP(&backupOutputPath, "output", "o", "", "Optional path to save the backup zip file")
	settingsRestoreCmd.Flags().StringVarP(&restoreInputPath, "input", "i", "", "Path to backup zip file to restore")
	settingsCmd.AddCommand(settingsBackupCmd)
	settingsCmd.AddCommand(settingsRestoreCmd)
	settingsCmd.AddCommand(settingsGetCmd)
	settingsGetCmd.Flags().BoolVar(&getJSON, "json", false, "Output raw JSON response")
	rootCmd.AddCommand(settingsCmd)
}

var restoreInputPath string

var settingsRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore server settings from a backup zip file",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		if restoreInputPath == "" {
			fmt.Fprintln(os.Stderr, "❌ --input is required")
			os.Exit(1)
		}

		file, err := os.Open(restoreInputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not open file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", filepath.Base(restoreInputPath))
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create form file: %v\n", err)
			os.Exit(1)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to copy file data: %v\n", err)
			os.Exit(1)
		}
		writer.Close()

		url := fmt.Sprintf("%s/api/settings/restore?token=%s&blockLists=true&logs=true&scopes=true&stats=true&zones=true&allowedZones=true&blockedZones=true&dnsSettings=true&logSettings=true&deleteExistingFiles=true&authConfig=true", host, token)
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "❌ Restore failed: HTTP %d\n", resp.StatusCode)
			os.Exit(1)
		}

		var result map[string]interface{}
		if getJSON {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to parse response: %v\n", err)
			os.Exit(1)
		}

		status, ok := result["status"].(string)
		if !ok || status != "ok" {
			if msg, ok := result["errorMessage"].(string); ok {
				fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
			} else {
				fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
			}
			os.Exit(1)
		}

		response := result["response"].(map[string]interface{})

		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()

		fmt.Println(bold("DNS Server Info:"))
		fmt.Printf("  Domain: %s\n", cyan(response["dnsServerDomain"]))
		fmt.Printf("  Version: %s\n", green(response["version"]))
		fmt.Printf("  Started: %s\n", response["uptimestamp"])
		fmt.Println()

		fmt.Println(bold("Network:"))
		fmt.Printf("  Endpoints: %v\n", response["dnsServerLocalEndPoints"])
		fmt.Printf("  IPv4 Sources: %v\n", response["dnsServerIPv4SourceAddresses"])
		fmt.Printf("  IPv6 Sources: %v\n", response["dnsServerIPv6SourceAddresses"])
		fmt.Println()

		fmt.Println(bold("Cache & Resolver:"))
		fmt.Printf("  Save Cache: %v\n", response["saveCache"])
		fmt.Printf("  Serve Stale: %v\n", response["serveStale"])
		fmt.Printf("  Prefetch Trigger: %v\n", response["cachePrefetchTrigger"])
		fmt.Println()

		fmt.Println(bold("Blocking:"))
		fmt.Printf("  Enabled: %v\n", response["enableBlocking"])
		fmt.Printf("  Custom Addresses: %v\n", response["customBlockingAddresses"])
		fmt.Println()

		fmt.Println(bold("Web Service:"))
		fmt.Printf("  HTTP Port: %v\n", response["webServiceHttpPort"])
		fmt.Printf("  TLS Port: %v\n", response["webServiceTlsPort"])
		fmt.Printf("  Enable TLS: %v\n", response["webServiceEnableTls"])

	},
}

var settingsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve current server settings",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		url := fmt.Sprintf("%s/api/settings/get?token=%s", host, token)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("❌ Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "❌ Failed: HTTP %d\n", resp.StatusCode)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to parse response: %v\n", err)
			os.Exit(1)
		}

		if getJSON {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))

			return
		}

		response := result["response"].(map[string]interface{})

		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		//gray := color.New(color.FgHiBlack).SprintFunc()

		fmt.Println(bold("General Settings:"))
		fmt.Printf("  Version: %s\n", green(response["version"]))
		fmt.Printf("  Start Time: %s\n", response["uptimestamp"])
		fmt.Printf("  Domain: %s\n", cyan(response["dnsServerDomain"]))
		fmt.Println()

		fmt.Println(bold("DNS Endpoints:"))
		fmt.Printf("  Local: %v\n", response["dnsServerLocalEndPoints"])
		fmt.Printf("  IPv4: %v\n", response["dnsServerIPv4SourceAddresses"])
		fmt.Printf("  IPv6: %v\n", response["dnsServerIPv6SourceAddresses"])
		fmt.Println()

		fmt.Println(bold("Blocking:"))
		fmt.Printf("  Enabled: %v\n", green(response["enableBlocking"]))
		fmt.Printf("  Type: %s\n", response["blockingType"])
		fmt.Printf("  TTL: %v\n", response["blockingAnswerTtl"])
		fmt.Printf("  Custom Addresses: %v\n", response["customBlockingAddresses"])
		fmt.Println()

		fmt.Println(bold("DNSSEC & Cache:"))
		fmt.Printf("  DNSSEC: %v\n", response["dnssecValidation"])
		fmt.Printf("  Save Cache: %v\n", response["saveCache"])
		fmt.Printf("  Serve Stale: %v\n", response["serveStale"])
		fmt.Printf("  Max Entries: %v\n", response["cacheMaximumEntries"])
		fmt.Printf("  Failure TTL: %v\n", response["cacheFailureRecordTtl"])
		fmt.Println()

		fmt.Println(bold("Forwarders:"))
		fmt.Printf("  Enabled: %v\n", response["concurrentForwarding"])
		fmt.Printf("  Protocol: %v\n", response["forwarderProtocol"])
		fmt.Printf("  Timeout: %vms\n", response["forwarderTimeout"])
		fmt.Println()

		fmt.Println(bold("Web Service:"))
		fmt.Printf("  HTTP Port: %v\n", response["webServiceHttpPort"])
		fmt.Printf("  TLS Port: %v\n", response["webServiceTlsPort"])
		fmt.Printf("  TLS Enabled: %v\n", response["webServiceEnableTls"])
		fmt.Println()

		fmt.Println(bold("Stats & Logging:"))
		fmt.Printf("  Enable Logging: %v\n", response["enableLogging"])
		fmt.Printf("  Log Folder: %v\n", response["logFolder"])
		fmt.Printf("  In-Memory Stats: %v\n", response["enableInMemoryStats"])
		fmt.Printf("  Max Log Days: %v\n", response["maxLogFileDays"])
		fmt.Println()

		fmt.Println(bold("TSIG Keys:"))
		tsigKeys := response["tsigKeys"].([]interface{})
		for _, entry := range tsigKeys {
			key := entry.(map[string]interface{})
			fmt.Printf("  - %s (%s)\n", cyan(key["keyName"]), yellow(key["algorithmName"]))
		}
	},
}
