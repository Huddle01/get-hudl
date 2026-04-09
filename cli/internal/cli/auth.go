package cli

import (
	"fmt"
	"strings"

	"github.com/Huddle01/hudl/cli/internal/config"
	"github.com/Huddle01/hudl/cli/internal/runtime"
	"github.com/spf13/cobra"
)

func newLoginCommand() *cobra.Command {
	var token string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an API key for future commands",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			if strings.TrimSpace(token) == "" && app.IsTTYIn {
				value, err := runtime.PromptString(app.Stdin, app.Stderr, "API key", "", true)
				if err != nil {
					return renderError(app, err)
				}
				token = value
			}
			token = strings.TrimSpace(token)
			if token == "" {
				return renderError(app, fmt.Errorf("token is required; pass --token"))
			}
			if err := config.SaveUserConfig(func(cfg *config.File) error {
				cfg.APIKey = token
				if cfg.API.CloudBaseURL == "" {
					cfg.API.CloudBaseURL = app.Config.CloudBase
				}
				if cfg.API.GPUBaseURL == "" {
					cfg.API.GPUBaseURL = app.Config.GPUBase
				}
				return nil
			}); err != nil {
				return renderError(app, err)
			}

			return executeResult(app, map[string]any{
				"ok":        true,
				"api_key":   maskToken(token),
				"user_path": app.Config.UserPath,
			}, nil)
		},
	}
	cmd.Flags().StringVar(&token, "token", "", "API key to save")
	return cmd
}

func newAuthCommand() *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Inspect and clear local authentication state",
	}
	auth.AddCommand(
		&cobra.Command{
			Use:   "status",
			Short: "Show the current authentication state",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app := appFromCommand(cmd)
				return executeResult(app, map[string]any{
					"configured": app.Config.APIKey != "",
					"api_key":    maskToken(app.Config.APIKey),
					"workspace":  app.Config.Workspace,
					"region":     app.Config.Region,
					"user_path":  app.Config.UserPath,
				}, nil)
			},
		},
		&cobra.Command{
			Use:   "clear",
			Short: "Remove the saved API key from the user config",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app := appFromCommand(cmd)
				if err := config.ClearUserAuth(); err != nil {
					return renderError(app, err)
				}
				return executeResult(app, map[string]any{"ok": true}, nil)
			},
		},
	)
	return auth
}

func newContextCommand() *cobra.Command {
	ctxCmd := &cobra.Command{
		Use:   "ctx",
		Short: "Inspect and update workspace/region defaults",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			return executeResult(app, map[string]any{
				"workspace":    app.Config.Workspace,
				"region":       app.Config.Region,
				"project_path": app.Config.ProjectPath,
				"user_path":    app.Config.UserPath,
			}, nil)
		},
	}

	ctxCmd.AddCommand(
		&cobra.Command{
			Use:   "use <workspace>",
			Short: "Set the default workspace in the user config",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app := appFromCommand(cmd)
				if err := config.SaveUserConfig(func(cfg *config.File) error {
					cfg.Workspace = args[0]
					return nil
				}); err != nil {
					return renderError(app, err)
				}
				return executeResult(app, map[string]any{"workspace": args[0]}, nil)
			},
		},
		&cobra.Command{
			Use:   "region <region>",
			Short: "Set the default region in the user config",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app := appFromCommand(cmd)
				if err := config.SaveUserConfig(func(cfg *config.File) error {
					cfg.Region = args[0]
					return nil
				}); err != nil {
					return renderError(app, err)
				}
				return executeResult(app, map[string]any{"region": args[0]}, nil)
			},
		},
	)
	return ctxCmd
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
