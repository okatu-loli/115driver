package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/SheltonZhu/115driver/cli/internal/auth"
	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/spf13/cobra"
)

var loginCookie string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with 115 cloud storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginCookie != "" {
			return loginWithCookie()
		}
		return loginWithQRCode()
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginCookie, "cookie", "", "Cookie string for authentication")
	rootCmd.AddCommand(loginCmd)
}

func loginWithCookie() error {
	cr := &driver.Credential{}
	if err := cr.FromCookie(loginCookie); err != nil {
		return &exitError{code: output.ExitArgs, msg: fmt.Sprintf("Invalid cookie: %v", err)}
	}

	c := driver.New(driver.UA(driver.UA115Browser)).ImportCredential(cr)
	if err := c.LoginCheck(); err != nil {
		return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Cookie validation failed: %v", err)}
	}

	if err := auth.SaveCredential(configPath, profile, loginCookie); err != nil {
		return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to save config: %v", err)}
	}

	printer.PrintSuccess(map[string]interface{}{
		"profile":      auth.DefaultProfile,
		"cookie_saved": true,
	})
	if !jsonOutput {
		fmt.Println("Login successful. Cookie saved to config.")
	}
	return nil
}

func loginWithQRCode() error {
	c := driver.New(driver.UA(driver.UA115Browser))
	session, err := c.QRCodeStart()
	if err != nil {
		return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to start QR login: %v", err)}
	}

	if jsonOutput {
		fmt.Fprintf(os.Stderr, `{"qr_url":"https://qrcodeapi.115.com/api/1.0/mac/1.0/qrcode?uid=%s","message":"Scan QR code with 115 app"}`+"\n", session.UID)
	} else {
		fmt.Fprintln(os.Stderr, "Scan the QR code with 115 app to login:")
		fmt.Fprintf(os.Stderr, "URL: https://qrcodeapi.115.com/api/1.0/mac/1.0/qrcode?uid=%s\n\n", session.UID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return &exitError{code: output.ExitError, msg: "QR code login timed out."}
		default:
		}

		time.Sleep(3 * time.Second)
		status, err := c.QRCodeStatus(session)
		if err != nil {
			continue
		}

		if status.IsExpired() {
			return &exitError{code: output.ExitError, msg: "QR code expired. Please try again."}
		}
		if status.IsCanceled() {
			return &exitError{code: output.ExitError, msg: "QR code login canceled."}
		}
		if status.IsAllowed() {
			cred, err := c.QRCodeLogin(session)
			if err != nil {
				return &exitError{code: output.ExitAuth, msg: fmt.Sprintf("Login failed: %v", err)}
			}

			cookieStr := cred.Cookie()
			if err := auth.SaveCredential(configPath, profile, cookieStr); err != nil {
				return &exitError{code: output.ExitError, msg: fmt.Sprintf("Failed to save config: %v", err)}
			}

			if jsonOutput {
				printer.PrintSuccess(map[string]interface{}{
					"profile":      auth.DefaultProfile,
					"cookie_saved": true,
				})
			} else {
				fmt.Println("\nLogin successful. Cookie saved to config.")
			}
			return nil
		}
		if !jsonOutput {
			fmt.Fprint(os.Stderr, ".")
		}
	}
}
