package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"tdns/internal/api"
)

// Name & Version of the app
var (
	Name    = "tdns"
	Version string
)

var (
	debugFlag bool
	ctx       context.Context
)
var defaultHost = "http://localhost:5380"
var defaultToken = ""

var rootCmd = &cobra.Command{
	Use:           fmt.Sprintf("%s", Name),
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         fmt.Sprintf("%s is a CLI tool for managing DNS zones", Name),
	Long:          fmt.Sprintf("%s is a CLI tool to manage Technitium DNS server via API endpoint", Name),
	//Run: func(cmd *cobra.Command, args []string) {
	//	_ = cmd.Help()
	//},
}

func Execute(version string) {
	Version = version

	rootCmd.Version = version

	cobra.CheckErr(rootCmd.Execute())

	if debugFlag {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	slog.Debug(fmt.Sprintf("App version: %s", Version))
}

func init() {
	cobra.OnInitialize(initConfig)

	// start with empty Context
	ctx = context.Background()

	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Enable debugging logging")
	rootCmd.PersistentFlags().StringP("token", "t", "", "API token (overrides config/TDNS_API_TOKEN env)")
	rootCmd.PersistentFlags().StringP("endpoint", "e", "", "API endpoint (overrides config)")
	rootCmd.PersistentFlags().BoolP("legacy-token", "L", false, "Also send token as a query parameter (for older servers/endpoints that don't honor Authorization: Bearer)")
	rootCmd.PersistentFlags().DurationP("timeout", "T", api.DefaultTimeout, "API request timeout (e.g. 5s, 1m)")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("endpoint"))
	viper.BindPFlag("legacy_token", rootCmd.PersistentFlags().Lookup("legacy-token"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))

	viper.SetEnvPrefix("TDNS")
	viper.AutomaticEnv()

	viper.SetDefault("token", defaultToken)
	viper.SetDefault("host", defaultHost)

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.tdns")
	viper.AddConfigPath("/etc/tdns/")
}

func initConfig() {
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: config not found/loaded: %v\n", err)
	}
}
