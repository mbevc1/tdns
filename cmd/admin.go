package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"tdns/internal/api"
)

var adminCmd = &cobra.Command{
	Use:     "admin",
	Aliases: []string{"ad"},
	Short:   "Administrative commands",
}

var sessionID string
var createTokenUser string
var createTokenName string
var getUser string
var newPassword string
var currentPassword string
var totpCode string
var pbkdf2Iterations int
var interactive bool
var JSON bool

var listSessionsCmd = &cobra.Command{
	Use:         "list-sessions",
	Aliases:     []string{"ls"},
	Short:       "List active sessions",
	Annotations: map[string]string{"group": "Session Management"},
	Run: func(cmd *cobra.Command, args []string) {
		_, response, err := api.New().GetJSON("/api/admin/sessions/list", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		sessions, ok := response["sessions"].([]interface{})
		if !ok || len(sessions) == 0 {
			fmt.Println("No active sessions found.")
			return
		}

		bold := color.New(color.Bold).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Println(bold("Active Sessions:"))
		for _, s := range sessions {
			session := s.(map[string]interface{})
			current := session["isCurrentSession"].(bool)
			colorize := green
			if !current {
				colorize = yellow
			}

			t := session["lastSeen"].(string)
			parsed, _ := time.Parse(time.RFC3339, t)

			fmt.Printf("- %s (%s)\n", colorize(session["partialToken"]), session["type"])
			fmt.Printf("  User: %s\n", cyan(session["username"]))
			fmt.Printf("  Name: %v\n", session["tokenName"])
			fmt.Printf("  Seen: %s from %s\n", parsed.Local().Format("2006-01-02 15:04:05"), session["lastSeenRemoteAddress"])
			fmt.Printf("  Agent: %s\n", session["lastSeenUserAgent"])
		}
	},
}

var deleteSessionCmd = &cobra.Command{
	Use:         "delete-session",
	Aliases:     []string{"ds"},
	Short:       "Delete a session using its partial token",
	Annotations: map[string]string{"group": "Session Management"},
	Run: func(cmd *cobra.Command, args []string) {
		if sessionID == "" {
			fmt.Println("Error: --id flag is required")
			os.Exit(1)
		}

		fmt.Print("Are you sure you want to delete this session? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("❌ Aborted.")
			return
		}

		q := url.Values{"partialToken": {sessionID}}
		if _, _, err := api.New().GetJSON("/api/admin/sessions/delete", q); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Session '%s' deleted successfully.\n", sessionID)
	},
}

var createTokenCmd = &cobra.Command{
	Use:         "create-token",
	Aliases:     []string{"ct"},
	Short:       "Create a new API token for a user",
	Annotations: map[string]string{"group": "Session Management"},
	Run: func(cmd *cobra.Command, args []string) {
		if createTokenUser == "" || createTokenName == "" {
			fmt.Println("Error: --user and --token-name are required")
			os.Exit(1)
		}

		q := url.Values{"user": {createTokenUser}, "tokenName": {createTokenName}}
		_, response, err := api.New().GetJSON("/api/admin/sessions/createToken", q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Token created successfully:")
		fmt.Printf("  Username: %s\n", response["username"])
		fmt.Printf("  Token Name: %s\n", response["tokenName"])
		fmt.Printf("  Token: %s\n", response["token"])
	},
}

var adminListUsersCmd = &cobra.Command{
	Use:     "list-users",
	Aliases: []string{"lu"},
	Short:   "List all system users",
	Run: func(cmd *cobra.Command, args []string) {
		_, response, err := api.New().GetJSON("/api/admin/users/list", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		users, ok := response["users"].([]interface{})
		if !ok || len(users) == 0 {
			fmt.Println("No users found.")
			return
		}

		bold := color.New(color.Bold).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		blue := color.New(color.FgBlue).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()

		fmt.Println(bold("User List:"))
		for _, u := range users {
			user := u.(map[string]interface{})
			status := green("Enabled")
			if user["disabled"].(bool) {
				status = red("Disabled")
			}

			fmt.Printf("- %s (%s)\n", bold(user["displayName"]), blue(user["username"]))
			fmt.Printf("  Status: %s\n", status)
			fmt.Printf("  Previous Session: %s from %s\n", user["previousSessionLoggedOn"], user["previousSessionRemoteAddress"])
			fmt.Printf("  Recent Session: %s from %s\n", user["recentSessionLoggedOn"], user["recentSessionRemoteAddress"])
			fmt.Println()
		}
	},
}

var adminGetUserCmd = &cobra.Command{
	Use:     "get-user",
	Aliases: []string{"gu"},
	Short:   "Get details for a specific user",
	Run: func(cmd *cobra.Command, args []string) {
		if getUser == "" {
			fmt.Fprintln(os.Stderr, "❌ --user is required")
			os.Exit(1)
		}

		q := url.Values{"user": {getUser}, "includeGroups": {"true"}}
		_, user, err := api.New().GetJSON("/api/admin/users/get", q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		bold := color.New(color.Bold).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		blue := color.New(color.FgBlue).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()

		fmt.Printf("%s (%s)\n", bold(user["displayName"]), blue(user["username"]))
		status := green("Enabled")
		if user["disabled"].(bool) {
			status = red("Disabled")
		}
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Groups: %v\n", user["groups"])
		fmt.Printf("  Session Timeout: %v seconds\n", user["sessionTimeoutSeconds"])
		fmt.Printf("  Previous Login: %s from %s\n", user["previousSessionLoggedOn"], user["previousSessionRemoteAddress"])
		fmt.Printf("  Recent Login: %s from %s\n", user["recentSessionLoggedOn"], user["recentSessionRemoteAddress"])
		fmt.Println()
		if sessions, ok := user["sessions"].([]interface{}); ok && len(sessions) > 0 {
			fmt.Println(bold("Sessions:"))
			for _, s := range sessions {
				session := s.(map[string]interface{})
				fmt.Printf("- Token: %s (%s)\n", blue(session["partialToken"]), session["type"])
				fmt.Printf("  Seen: %s from %s\n", session["lastSeen"], session["lastSeenRemoteAddress"])
				fmt.Printf("  Agent: %s\n", session["lastSeenUserAgent"])
			}
		}
	},
}

var adminCheckUpdateCmd = &cobra.Command{
	Use:     "check-update",
	Aliases: []string{"cu"},
	Short:   "Check for available updates",
	Run: func(cmd *cobra.Command, args []string) {
		result, respData, err := api.New().GetJSON("/api/user/checkForUpdate", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if JSON {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}

		bold := color.New(color.Bold).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Printf("%s: %s -> %s\n", bold("Version"), cyan(respData["currentVersion"]), green(respData["updateVersion"]))

		if respData["updateAvailable"].(bool) {
			fmt.Printf("⚠️  %s\n", bold(red(respData["updateTitle"])))
			fmt.Printf("%s\n\n", respData["updateMessage"])
			fmt.Printf("Download: %s\n", cyan(respData["downloadLink"]))
			fmt.Printf("Instructions: %s\n", cyan(respData["instructionsLink"]))
			fmt.Printf("Changelog: %s\n", cyan(respData["changeLogLink"]))
		} else {
			fmt.Println("✅ You are using the latest version.")
		}
	},
}

var adminChangePasswordCmd = &cobra.Command{
	Use:     "change-password",
	Aliases: []string{"cp"},
	Short:   "Change the current user's password",
	Run: func(cmd *cobra.Command, args []string) {
		if interactive {
			if currentPassword == "" {
				fmt.Print("Enter current password: ")
				byteCurrent, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Failed to read password: %v\n", err)
					os.Exit(1)
				}
				currentPassword = string(byteCurrent)
			}

			if newPassword == "" {
				fmt.Print("Enter new password: ")
				bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Failed to read password: %v\n", err)
					os.Exit(1)
				}

				fmt.Print("Confirm new password: ")
				byteConfirm, _ := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if string(byteConfirm) != string(bytePassword) {
					fmt.Fprintln(os.Stderr, "❌ Passwords do not match")
					os.Exit(1)
				}

				newPassword = string(bytePassword)
			}
		}

		if currentPassword == "" {
			fmt.Fprintln(os.Stderr, "❌ --current is required")
			os.Exit(1)
		}

		if newPassword == "" {
			fmt.Fprintln(os.Stderr, "❌ --new is required")
			os.Exit(1)
		}

		params := url.Values{
			"pass":    {currentPassword},
			"newPass": {newPassword},
		}
		if totpCode != "" {
			params.Set("totp", totpCode)
		}
		if pbkdf2Iterations > 0 {
			params.Set("iterations", fmt.Sprintf("%d", pbkdf2Iterations))
		}

		if _, _, err := api.New().GetJSON("/api/user/changePassword", params); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Password changed successfully.")
	},
}

func init() {
	adminChangePasswordCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Prompt for password interactively")
	adminChangePasswordCmd.Flags().StringVarP(&currentPassword, "current", "c", "", "Current password (insecure)")
	adminChangePasswordCmd.Flags().StringVarP(&newPassword, "new", "n", "", "New password (insecure)")
	adminChangePasswordCmd.Flags().StringVarP(&totpCode, "totp", "o", "", "6-digit TOTP code if 2FA is enabled")
	adminChangePasswordCmd.Flags().IntVar(&pbkdf2Iterations, "iterations", 0, "Number of iterations for PBKDF2 SHA256 password hashing")
	adminCmd.AddCommand(adminChangePasswordCmd)
	adminCmd.AddCommand(adminCheckUpdateCmd)
	adminCheckUpdateCmd.Flags().BoolVar(&JSON, "json", false, "Output raw JSON response")

	adminGetUserCmd.Flags().StringVarP(&getUser, "user", "u", "", "User to query")
	adminCmd.AddCommand(adminGetUserCmd)

	adminCmd.AddCommand(adminListUsersCmd)
	deleteSessionCmd.Flags().StringVarP(&sessionID, "id", "i", "", "Partial token of the session to delete")
	createTokenCmd.Flags().StringVar(&createTokenUser, "user", "", "User to create token for")
	createTokenCmd.Flags().StringVar(&createTokenName, "token-name", "", "Name for the new token")
	adminCmd.AddCommand(deleteSessionCmd)
	adminCmd.AddCommand(createTokenCmd)
	adminCmd.AddCommand(listSessionsCmd)
	rootCmd.AddCommand(adminCmd)
}
