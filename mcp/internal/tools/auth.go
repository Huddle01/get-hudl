package tools

import (
	"fmt"
	"strings"

	"github.com/Huddle01/get-hudl/internal/config"
	"github.com/Huddle01/get-hudl/mcp/internal/server"
)

func registerAuthTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_auth_status",
		Description: "Show the current authentication state — API key, workspace, region, and config path. Use this to verify the user is logged in before making API calls.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		// Always re-read config for status check
		app, err := reloadApp()
		if err != nil {
			return nil, err
		}
		maskedCloud := maskToken(app.Config.APIKey)
		maskedGPU := maskToken(app.Config.GPUAPIKey)
		return map[string]any{
			"configured":  app.Config.APIKey != "" || app.Config.GPUAPIKey != "",
			"api_key":     maskedCloud,
			"gpu_api_key": maskedGPU,
			"workspace":  app.Config.Workspace,
			"region":     app.Config.Region,
			"user_path":  app.Config.UserPath,
		}, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_login",
		Description: "Store API keys for authenticating with Huddle01. Pass token for Cloud API key, gpu_token for GPU API key, or both.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"token":     server.StringProp("The Cloud API key to store"),
			"gpu_token": server.StringProp("The GPU API key to store"),
		}, nil),
	}, func(args map[string]any) (any, error) {
		token := strings.TrimSpace(server.ArgString(args, "token"))
		gpuToken := strings.TrimSpace(server.ArgString(args, "gpu_token"))
		if token == "" && gpuToken == "" {
			return nil, fmt.Errorf("at least one of token or gpu_token is required")
		}
		if err := config.SaveUserConfig(func(cfg *config.File) error {
			if token != "" {
				cfg.APIKey = token
			}
			if gpuToken != "" {
				cfg.GPUAPIKey = gpuToken
			}
			return nil
		}); err != nil {
			return nil, err
		}
		invalidateCache()
		result := map[string]any{"ok": true}
		if token != "" {
			result["api_key"] = maskToken(token)
		}
		if gpuToken != "" {
			result["gpu_api_key"] = maskToken(gpuToken)
		}
		return result, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_auth_clear",
		Description: "Remove the saved API key from the user config. Subsequent API calls will fail until a new key is stored with hudl_login.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		if err := config.ClearUserAuth(); err != nil {
			return nil, err
		}
		invalidateCache()
		return map[string]any{"ok": true}, nil
	})
}

func registerContextTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_ctx_show",
		Description: "Show the current workspace and region context. Returns the active workspace, region, and config file paths.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		app, err := reloadApp()
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"workspace":    app.Config.Workspace,
			"region":       app.Config.Region,
			"project_path": app.Config.ProjectPath,
			"user_path":    app.Config.UserPath,
		}, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_ctx_use",
		Description: "Set the default workspace in the user config. All subsequent API calls will use this workspace.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"workspace": server.StringProp("Workspace name to set as default"),
		}, []string{"workspace"}),
	}, func(args map[string]any) (any, error) {
		ws := server.ArgString(args, "workspace")
		if ws == "" {
			return nil, fmt.Errorf("workspace is required")
		}
		if err := config.SaveUserConfig(func(cfg *config.File) error {
			cfg.Workspace = ws
			return nil
		}); err != nil {
			return nil, err
		}
		invalidateCache()
		return map[string]any{"workspace": ws}, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_ctx_region",
		Description: "Set the default region in the user config. All subsequent cloud API calls will target this region.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"region": server.StringProp("Region code to set as default (e.g. eu2, us1)"),
		}, []string{"region"}),
	}, func(args map[string]any) (any, error) {
		region := server.ArgString(args, "region")
		if region == "" {
			return nil, fmt.Errorf("region is required")
		}
		if err := config.SaveUserConfig(func(cfg *config.File) error {
			cfg.Region = region
			return nil
		}); err != nil {
			return nil, err
		}
		invalidateCache()
		return map[string]any{"region": region}, nil
	})
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
