package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/calyptia/cloud-cli/auth0"
	"github.com/spf13/cobra"
)

// TODO: change it to project token authorization instead of user based authentication.
func newCmdLogin(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login by authorizing this CLI with Calyptia Cloud through a browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := config.auth0.DeviceCode(config.ctx)
			if err != nil {
				return fmt.Errorf("could not request a device code to login: %w", err)
			}

			fmt.Printf("Please visit the following link to authorize this CLI to use your Calyptia Cloud account:\n\n%s\n\nWaiting authorization...\n", code.VerificationURIComplete)

			for {
				time.Sleep(time.Second * time.Duration(code.Interval))
				tok, err := config.auth0.AccessToken(config.ctx, code.DeviceCode)
				if auth0.IsAuthorizationPendingError(err) {
					continue
				}

				if err != nil {
					return fmt.Errorf("could not authenticate you: %w", err)
				}

				if tok.RefreshToken == "" {
					return errors.New("oops: missing refresh token")
				}

				err = saveToken(tok)
				if err != nil {
					return fmt.Errorf("could not save your login: %w", err)
				}

				fmt.Println("\nSuccess! You are now authenticated.\nYour login information has been already stored and you won't need to run `calyptia login` again. Subsequent commands will use this auth info.")

				break
			}

			return nil
		},
	}
}
