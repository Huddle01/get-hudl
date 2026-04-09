package cli

import (
	"fmt"
	"strings"

	"github.com/Huddle01/get-hudl/cli/internal/config"
	"github.com/Huddle01/get-hudl/cli/internal/runtime"
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

			masked := maskToken(token)
			if app.IsTTYOut && outputMode(app) == "table" {
				fmt.Fprintf(app.Stdout, "Logged in. API key: %s\n", masked)
				fmt.Fprintf(app.Stdout, "Config saved to %s\n", app.Config.UserPath)
				return nil
			}
			return executeResult(app, map[string]any{
				"ok":        true,
				"api_key":   masked,
				"user_path": app.Config.UserPath,
			}, nil)
		},
	}
	cmd.Flags().StringVar(&token, "token", "", "API key to save")
	return cmd
}

func authStatusRunE(cmd *cobra.Command, _ []string) error {
	app := appFromCommand(cmd)
	configured := app.Config.APIKey != ""
	masked := maskToken(app.Config.APIKey)

	if app.IsTTYOut && outputMode(app) == "table" {
		if configured {
			fmt.Fprintf(app.Stdout, "Logged in\n")
			fmt.Fprintf(app.Stdout, "  API key:    %s\n", masked)
		} else {
			fmt.Fprintf(app.Stdout, "Not logged in\n")
			fmt.Fprintf(app.Stdout, "  Run: hudl login --token <key>\n")
		}
		if app.Config.Workspace != "" {
			fmt.Fprintf(app.Stdout, "  Workspace:  %s\n", app.Config.Workspace)
		}
		if app.Config.Region != "" {
			fmt.Fprintf(app.Stdout, "  Region:     %s\n", app.Config.Region)
		}
		fmt.Fprintf(app.Stdout, "  Config:     %s\n", app.Config.UserPath)
		return nil
	}

	return executeResult(app, map[string]any{
		"configured": configured,
		"api_key":    masked,
		"workspace":  app.Config.Workspace,
		"region":     app.Config.Region,
		"user_path":  app.Config.UserPath,
	}, nil)
}

func newAuthCommand() *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Inspect and clear local authentication state",
		RunE:  authStatusRunE,
	}

	auth.AddCommand(
		&cobra.Command{
			Use:   "status",
			Short: "Show the current authentication state",
			RunE:  authStatusRunE,
		},
		&cobra.Command{
			Use:   "clear",
			Short: "Remove the saved API key from the user config",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app := appFromCommand(cmd)
				if err := config.ClearUserAuth(); err != nil {
					return renderError(app, err)
				}
				if app.IsTTYOut && outputMode(app) == "table" {
					fmt.Fprintln(app.Stdout, "API key removed.")
					return nil
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
			if app.IsTTYOut && outputMode(app) == "table" {
				if app.Config.Workspace != "" {
					fmt.Fprintf(app.Stdout, "Workspace:  %s\n", app.Config.Workspace)
				} else {
					fmt.Fprintf(app.Stdout, "Workspace:  (not set)\n")
				}
				if app.Config.Region != "" {
					fmt.Fprintf(app.Stdout, "Region:     %s\n", app.Config.Region)
				} else {
					fmt.Fprintf(app.Stdout, "Region:     (not set)\n")
				}
				if app.Config.ProjectPath != "" {
					fmt.Fprintf(app.Stdout, "Project:    %s\n", app.Config.ProjectPath)
				}
				fmt.Fprintf(app.Stdout, "Config:     %s\n", app.Config.UserPath)
				return nil
			}
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
				if app.IsTTYOut && outputMode(app) == "table" {
					fmt.Fprintf(app.Stdout, "Workspace set to %s\n", args[0])
					return nil
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
				if app.IsTTYOut && outputMode(app) == "table" {
					fmt.Fprintf(app.Stdout, "Region set to %s\n", args[0])
					return nil
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
