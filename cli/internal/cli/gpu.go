package cli

import (
	"fmt"

	"github.com/Huddle01/get-hudl/internal/runtime"
	"github.com/spf13/cobra"
)

func newGPUCommand() *cobra.Command {
	gpu := &cobra.Command{Use: "gpu", Short: "Manage GPU marketplace and deployments"}
	gpu.AddCommand(
		newGPUOffersCommand(),
		newGPUSummaryCommand(),
		newGPUListCommand(),
		newGPUGetCommand(),
		newGPUDeployCommand(),
		newGPUActionCommand(),
		newGPUDeleteCommand(),
		newGPUCheckCommand(),
		newGPUWaitlistCommand(),
		newGPUImageCommand(),
		newGPUVolumeCommand(),
		newGPUSSHKeyCommand(),
		newGPUAPIKeyCommand(),
		newGPUWebhookCommand(),
		newGPURegionCommand(),
	)
	return gpu
}

func newGPUOffersCommand() *cobra.Command {
	var gpuModel, sortBy, sortOrder string
	var availableOnly bool = true
	var maxPrice, minVRAM string
	var limit, page int
	cmd := &cobra.Command{
		Use:   "offers",
		Short: "List available GPU offers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			query := map[string]string{}
			setQuery(query, "gpu_model", gpuModel)
			setQuery(query, "available_only", boolString(availableOnly))
			setQuery(query, "max_price", maxPrice)
			setQuery(query, "min_vram", minVRAM)
			setQuery(query, "sort_by", sortBy)
			setQuery(query, "sort_order", sortOrder)
			setQuery(query, "limit", intString(limit))
			setQuery(query, "page", intString(page))
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendGPU, Method: "GET", Path: "/gpus/available", Query: query,
			}, extractGPUList, runtime.MutateOptions{})
		},
	}
	cmd.Flags().StringVar(&gpuModel, "gpu-model", "", "Filter by GPU model")
	cmd.Flags().BoolVar(&availableOnly, "available-only", true, "Show only currently available offers")
	cmd.Flags().StringVar(&maxPrice, "max-price", "", "Maximum hourly price")
	cmd.Flags().StringVar(&minVRAM, "min-vram", "", "Minimum total VRAM")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field")
	cmd.Flags().StringVar(&sortOrder, "sort-order", "", "Sort order")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	return cmd
}

func newGPUSummaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "summary",
		Short: "Show GPU marketplace summary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendGPU, Method: "GET", Path: "/gpus/available/summary",
			}, extractGPUData, runtime.MutateOptions{})
		},
	}
}

func newGPUListCommand() *cobra.Command {
	var status, cursor string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List GPU deployments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			query := map[string]string{}
			setQuery(query, "status", status)
			setQuery(query, "cursor", cursor)
			setQuery(query, "limit", intString(limit))
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendGPU, Method: "GET", Path: "/deployments", Query: query,
			}, extractGPUList, runtime.MutateOptions{})
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by deployment status")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
	return cmd
}

func newGPUGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a GPU deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendGPU, Method: "GET", Path: "/deployments/" + args[0],
			}, extractGPUData, runtime.MutateOptions{})
		},
	}
}

func newGPUDeployCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var clusterType, image, hostname, description, location string
	var sshKeyIDs []string
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a GPU cluster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			body = runtime.MergeRequest(cloneRequest(app.Config.Defaults.GPU), body)
			setMapString(body, "cluster_type", clusterType)
			setMapString(body, "image", image)
			setMapString(body, "hostname", hostname)
			setMapString(body, "description", description)
			setMapString(body, "location", location)
			setMapStringArray(body, "ssh_key_ids", sshKeyIDs)
			if mut.Interactive && app.IsTTYIn {
				promptGPUDeploy(app, body)
			}
			for _, key := range []string{"cluster_type", "image", "hostname", "location", "ssh_key_ids"} {
				if !hasValue(body[key]) {
					return renderError(app, fmt.Errorf("%s is required", key))
				}
			}
			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendGPU,
				Method:         "POST",
				Path:           "/deployments/clusters",
				Body:           body,
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&clusterType, "cluster-type", "", "Cluster type / offer slug")
	cmd.Flags().StringVar(&image, "image", "", "Image slug")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Deployment hostname")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&location, "location", "", "Location code")
	cmd.Flags().StringArrayVar(&sshKeyIDs, "ssh-key-id", nil, "SSH key ID (repeatable)")
	return cmd
}

func newGPUActionCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "action <id> <action>",
		Short: "Run an action on a GPU deployment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			body := map[string]any{"action": args[1]}
			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendGPU,
				Method:         "POST",
				Path:           "/deployments/" + args[0] + "/actions",
				Body:           body,
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newGPUDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a GPU deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			if err := ensureConfirmation(app, mut, "Delete GPU deployment "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendGPU,
				Method:         "DELETE",
				Path:           "/deployments/" + args[0],
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newGPUCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check <cluster-type>",
		Short: "Check GPU cluster availability",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendGPU, Method: "GET", Path: "/deployments/availability/" + args[0],
			}, extractGPUData, runtime.MutateOptions{})
		},
	}
}

func newGPUWaitlistCommand() *cobra.Command {
	waitlist := &cobra.Command{
		Use:   "waitlist",
		Short: "Manage GPU waitlist requests",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return newGPUWaitlistListCommand().RunE(cmd, nil)
		},
	}
	waitlist.AddCommand(newGPUWaitlistListCommand(), newGPUWaitlistAddCommand(), newGPUWaitlistCancelCommand())
	return waitlist
}

func newGPUWaitlistListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List waitlist requests", RunE: func(cmd *cobra.Command, _ []string) error {
		app := appFromCommand(cmd)
		extract := func(raw map[string]any) (any, *runtime.Paging, error) {
			return raw["data"], nil, nil
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/deployments/requests"}, extract, runtime.MutateOptions{})
	}}
}

func newGPUWaitlistAddCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var clusterType, location string
	var autoDeploy bool
	var image, hostname string
	var sshKeyIDs []string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a waitlist request",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			setMapString(body, "cluster_type", clusterType)
			setMapString(body, "location", location)
			body["auto_deploy"] = autoDeploy
			if autoDeploy {
				configMap := map[string]any{}
				setMapString(configMap, "image", image)
				setMapString(configMap, "hostname", hostname)
				setMapStringArray(configMap, "ssh_key_ids", sshKeyIDs)
				if len(configMap) > 0 {
					body["config"] = configMap
				}
			}
			if mut.Interactive && app.IsTTYIn {
				if !hasValue(body["cluster_type"]) {
					value, err := runtime.PromptString(app.Stdin, app.Stderr, "Cluster type", "", true)
					if err != nil {
						return renderError(app, err)
					}
					body["cluster_type"] = value
				}
			}
			if !hasValue(body["cluster_type"]) {
				return renderError(app, fmt.Errorf("cluster_type is required"))
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "POST", Path: "/deployments/requests", Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&clusterType, "cluster-type", "", "Cluster type to wait for")
	cmd.Flags().StringVar(&location, "location", "", "Preferred location")
	cmd.Flags().BoolVar(&autoDeploy, "auto-deploy", false, "Automatically deploy when capacity is available")
	cmd.Flags().StringVar(&image, "image", "", "Auto-deploy image")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Auto-deploy hostname")
	cmd.Flags().StringArrayVar(&sshKeyIDs, "ssh-key-id", nil, "Auto-deploy SSH key ID (repeatable)")
	return cmd
}

func newGPUWaitlistCancelCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a waitlist request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			if err := ensureConfirmation(app, mut, "Cancel waitlist request "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "DELETE", Path: "/deployments/requests/" + args[0], Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newGPUImageCommand() *cobra.Command {
	image := &cobra.Command{Use: "image", Short: "Manage GPU images"}
	var clusterType, imageType string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List GPU images",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			query := map[string]string{}
			setQuery(query, "cluster_type", clusterType)
			setQuery(query, "image_type", imageType)
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/images", Query: query}, extractGPUList, runtime.MutateOptions{})
		},
	}
	listCmd.Flags().StringVar(&clusterType, "cluster-type", "", "Filter by compatible cluster type")
	listCmd.Flags().StringVar(&imageType, "image-type", "", "Image type filter")
	image.AddCommand(listCmd)
	return image
}

func newGPUVolumeCommand() *cobra.Command {
	volume := &cobra.Command{Use: "volume", Short: "Manage GPU volumes"}
	var status string
	volume.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List GPU volumes",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app := appFromCommand(cmd)
				query := map[string]string{}
				setQuery(query, "status", status)
				return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/volumes", Query: query}, extractGPUList, runtime.MutateOptions{})
			},
		},
		newGPUVolumeCreateCommand(),
	)
	volume.PersistentFlags().StringVar(&status, "status", "", "Filter by volume status")
	return volume
}

func newGPUVolumeCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var volumeType, location string
	var size int
	var instanceID string
	var instanceIDs []string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a GPU volume",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			body = runtime.MergeRequest(cloneRequest(app.Config.Defaults.GPUVolume), body)
			if len(args) > 0 {
				body["name"] = args[0]
			}
			setMapString(body, "type", volumeType)
			setMapString(body, "location", location)
			setMapInt(body, "size", size)
			setMapString(body, "instance_id", instanceID)
			setMapStringArray(body, "instance_ids", instanceIDs)
			for _, key := range []string{"name", "type", "location", "size"} {
				if !hasValue(body[key]) {
					return renderError(app, fmt.Errorf("%s is required", key))
				}
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "POST", Path: "/volumes", Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&volumeType, "type", "", "Volume type")
	cmd.Flags().StringVar(&location, "location", "", "Location code")
	cmd.Flags().IntVar(&size, "size", 0, "Size in GB")
	cmd.Flags().StringVar(&instanceID, "instance-id", "", "Instance ID to attach to")
	cmd.Flags().StringArrayVar(&instanceIDs, "instance-ids", nil, "Instance IDs for shared volume")
	return cmd
}

func newGPUSSHKeyCommand() *cobra.Command {
	key := &cobra.Command{Use: "key", Short: "Manage GPU SSH keys"}
	key.AddCommand(
		&cobra.Command{Use: "list", Short: "List GPU SSH keys", RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			extract := func(raw map[string]any) (any, *runtime.Paging, error) { return raw["data"], nil, nil }
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/ssh-keys"}, extract, runtime.MutateOptions{})
		}},
		newGPUSSHKeyUploadCommand(),
		newGPUSSHKeyDeleteCommand(),
	)
	return key
}

func newGPUSSHKeyUploadCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var publicKey string
	cmd := &cobra.Command{
		Use:   "upload <name>",
		Short: "Upload a GPU SSH key",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			if len(args) > 0 {
				body["name"] = args[0]
			}
			setMapString(body, "public_key", publicKey)
			if !hasValue(body["name"]) || !hasValue(body["public_key"]) {
				return renderError(app, fmt.Errorf("name and public_key are required"))
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "POST", Path: "/ssh-keys", Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&publicKey, "public-key", "", "SSH public key")
	return cmd
}

func newGPUSSHKeyDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a GPU SSH key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			if err := ensureConfirmation(app, mut, "Delete GPU SSH key "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "DELETE", Path: "/ssh-keys/" + args[0], Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newGPUAPIKeyCommand() *cobra.Command {
	apiKey := &cobra.Command{Use: "apikey", Short: "Manage GPU API keys"}
	apiKey.AddCommand(
		&cobra.Command{Use: "list", Short: "List GPU API keys", RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			extract := func(raw map[string]any) (any, *runtime.Paging, error) { return raw["data"], nil, nil }
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/api-keys"}, extract, runtime.MutateOptions{})
		}},
		newGPUAPIKeyCreateCommand(),
		newGPUAPIKeyRevokeCommand(),
	)
	return apiKey
}

func newGPUAPIKeyCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a GPU API key",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			if len(args) > 0 {
				body["name"] = args[0]
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "POST", Path: "/api-keys", Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	return cmd
}

func newGPUAPIKeyRevokeCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a GPU API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			if err := ensureConfirmation(app, mut, "Revoke GPU API key "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "DELETE", Path: "/api-keys/" + args[0], Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newGPUWebhookCommand() *cobra.Command {
	webhook := &cobra.Command{Use: "webhook", Short: "Manage GPU webhooks"}
	webhook.AddCommand(newGPUWebhookListCommand(), newGPUWebhookCreateCommand(), newGPUWebhookUpdateCommand(), newGPUWebhookDeleteCommand())
	return webhook
}

func newGPUWebhookListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List webhooks", RunE: func(cmd *cobra.Command, _ []string) error {
		app := appFromCommand(cmd)
		extract := func(raw map[string]any) (any, *runtime.Paging, error) { return raw["data"], nil, nil }
		return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/webhooks"}, extract, runtime.MutateOptions{})
	}}
}

func newGPUWebhookCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var url string
	var events []string
	cmd := &cobra.Command{
		Use:   "create <url>",
		Short: "Create a webhook",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			body = runtime.MergeRequest(cloneRequest(app.Config.Defaults.GPUWebhook), body)
			if len(args) > 0 {
				body["url"] = args[0]
			}
			setMapString(body, "url", url)
			setMapStringArray(body, "events", events)
			if mut.Interactive && app.IsTTYIn {
				promptWebhook(app, body, true)
			}
			if !hasValue(body["url"]) || !hasValue(body["events"]) {
				return renderError(app, fmt.Errorf("url and events are required"))
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "POST", Path: "/webhooks", Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&url, "url", "", "Webhook URL")
	cmd.Flags().StringArrayVar(&events, "event", nil, "Event name (repeatable)")
	return cmd
}

func newGPUWebhookUpdateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var url string
	var events []string
	var isActive bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			setMapString(body, "url", url)
			setMapStringArray(body, "events", events)
			if cmd.Flags().Changed("active") {
				body["is_active"] = isActive
			}
			if mut.Interactive && app.IsTTYIn {
				promptWebhook(app, body, false)
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "PUT", Path: "/webhooks/" + args[0], Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractGPUData, mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&url, "url", "", "Webhook URL")
	cmd.Flags().StringArrayVar(&events, "event", nil, "Event name (repeatable)")
	cmd.Flags().BoolVar(&isActive, "active", true, "Set the webhook active state")
	return cmd
}

func newGPUWebhookDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			if err := ensureConfirmation(app, mut, "Delete webhook "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "DELETE", Path: "/webhooks/" + args[0], Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newGPURegionCommand() *cobra.Command {
	region := &cobra.Command{Use: "region", Short: "Discover GPU locations"}
	region.AddCommand(
		&cobra.Command{Use: "list", Short: "List GPU regions", RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			extract := func(raw map[string]any) (any, *runtime.Paging, error) { return raw["data"], nil, nil }
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/regions"}, extract, runtime.MutateOptions{})
		}},
		&cobra.Command{Use: "volume-types", Short: "List GPU volume types", RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			extract := func(raw map[string]any) (any, *runtime.Paging, error) { return raw["data"], nil, nil }
			return handleRequest(app, runtime.Request{Backend: runtime.BackendGPU, Method: "GET", Path: "/volume-types"}, extract, runtime.MutateOptions{})
		}},
	)
	return region
}

func promptGPUDeploy(app *runtime.App, body map[string]any) {
	if !hasValue(body["cluster_type"]) {
		if value, err := runtime.PromptString(app.Stdin, app.Stderr, "Cluster type", "", true); err == nil {
			body["cluster_type"] = value
		}
	}
	if !hasValue(body["image"]) {
		if value, err := runtime.PromptString(app.Stdin, app.Stderr, "Image", "", true); err == nil {
			body["image"] = value
		}
	}
	if !hasValue(body["hostname"]) {
		if value, err := runtime.PromptString(app.Stdin, app.Stderr, "Hostname", "", true); err == nil {
			body["hostname"] = value
		}
	}
	if !hasValue(body["location"]) {
		if value, err := runtime.PromptString(app.Stdin, app.Stderr, "Location", "", true); err == nil {
			body["location"] = value
		}
	}
	if len(stringList(body["ssh_key_ids"])) == 0 {
		if values, err := runtime.PromptCSV(app.Stdin, app.Stderr, "SSH key IDs (comma-separated)", nil, true); err == nil {
			body["ssh_key_ids"] = values
		}
	}
}

func promptWebhook(app *runtime.App, body map[string]any, requireEvents bool) {
	if !hasValue(body["url"]) {
		if value, err := runtime.PromptString(app.Stdin, app.Stderr, "Webhook URL", "", true); err == nil {
			body["url"] = value
		}
	}
	if requireEvents && !hasValue(body["events"]) {
		if values, err := runtime.PromptCSV(app.Stdin, app.Stderr, "Events (comma-separated)", nil, true); err == nil {
			body["events"] = values
		}
	}
}
