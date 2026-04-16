package cmd

import (
	"fmt"
	"os"

	"github.com/SheltonZhu/115driver/cli/internal/auth"
	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var (
	cookieFlag string
	configPath string
	profile    string
	jsonOutput bool
	debugMode  bool
)

var client *driver.Pan115Client
var printer *output.Printer

var rootCmd = &cobra.Command{
	Use:           "115driver",
	Short:         "CLI tool for 115 cloud storage",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       "0.1.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		printer = output.NewPrinter(jsonOutput)

		if cmd.Name() == "login" {
			return nil
		}

		cr, err := auth.ResolveCredential(cookieFlag, configPath, profile)
		if err != nil {
			return &exitError{code: output.ExitAuth, msg: err.Error()}
		}

		opts := []driver.Option{driver.UA(driver.UA115Browser)}
		if debugMode {
			opts = append(opts, driver.WithDebug())
		}
		client = driver.New(opts...).ImportCredential(cr)

		if _, err := client.GetUser(); err != nil {
			return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Authentication failed: %v\nRun '115driver login' to re-authenticate.", err)}
		}
		return nil
	},
}

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string {
	return e.msg
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cookieFlag, "cookie", "", "Cookie string (or set 115DRIVER_COOKIE env)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Config file path (default ~/.115driver/config.toml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Config profile name (default 'main')")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for AI agents)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug mode")
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		if ee, ok := err.(*exitError); ok {
			if printer != nil {
				return printer.PrintError(ee.msg, ee.code)
			}
			fmt.Fprintln(os.Stderr, ee.msg)
			return ee.code
		}
		if printer != nil {
			return printer.PrintError(err.Error(), output.ExitError)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return output.ExitError
	}
	return output.ExitSuccess
}
