package tools

import (
	"fmt"

	"github.com/Huddle01/get-hudl/mcp/internal/server"
)

func registerGPUTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_offers",
		Description: "List available GPU offers on the marketplace. Filter by model, price, VRAM, and availability.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"gpu_model":      server.StringProp("Filter by GPU model (e.g. A100, H100)"),
			"available_only": server.BoolProp("Show only currently available offers (default true)"),
			"max_price":      server.StringProp("Maximum hourly price filter"),
			"min_vram":       server.StringProp("Minimum total VRAM filter"),
			"sort_by":        server.StringProp("Field to sort by"),
			"sort_order":     server.EnumProp("Sort direction", []string{"asc", "desc"}),
			"limit":          server.IntProp("Results per page"),
			"page":           server.IntProp("Page number"),
		}, nil),
	}, func(args map[string]any) (any, error) {
		q := map[string]string{}
		setQuery(q, "gpu_model", server.ArgString(args, "gpu_model"))
		setQuery(q, "available_only", boolStr(server.ArgBool(args, "available_only", true)))
		setQuery(q, "max_price", server.ArgString(args, "max_price"))
		setQuery(q, "min_vram", server.ArgString(args, "min_vram"))
		setQuery(q, "sort_by", server.ArgString(args, "sort_by"))
		setQuery(q, "sort_order", server.ArgString(args, "sort_order"))
		setQuery(q, "limit", intStr(server.ArgInt(args, "limit")))
		setQuery(q, "page", intStr(server.ArgInt(args, "page")))
		raw, err := gpuRequest("GET", "/gpus/available", q, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractGPUListWithMeta(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_summary",
		Description: "Show GPU marketplace summary — aggregate availability across all models.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/gpus/available/summary", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_list",
		Description: "List your GPU deployments.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"status": server.StringProp("Filter by deployment status"),
			"cursor": server.StringProp("Pagination cursor"),
			"limit":  server.IntProp("Results per page"),
		}, nil),
	}, func(args map[string]any) (any, error) {
		q := map[string]string{}
		setQuery(q, "status", server.ArgString(args, "status"))
		setQuery(q, "cursor", server.ArgString(args, "cursor"))
		setQuery(q, "limit", intStr(server.ArgInt(args, "limit")))
		raw, err := gpuRequest("GET", "/deployments", q, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractGPUListWithMeta(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_get",
		Description: "Get details of a GPU deployment.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Deployment ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("GET", "/deployments/"+id, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_deploy",
		Description: "Deploy a new GPU cluster. Requires cluster_type, image, hostname, location, and ssh_key_ids.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"cluster_type": server.StringProp("Cluster type / offer slug (e.g. a100-single)"),
			"image":        server.StringProp("Image slug"),
			"hostname":     server.StringProp("Deployment hostname"),
			"description":  server.StringProp("Description (optional)"),
			"location":     server.StringProp("Location code"),
			"ssh_key_ids":  server.StringArrayProp("SSH key IDs"),
		}, []string{"cluster_type", "image", "hostname", "location", "ssh_key_ids"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"cluster_type": server.ArgString(args, "cluster_type"),
			"image":        server.ArgString(args, "image"),
			"hostname":     server.ArgString(args, "hostname"),
			"location":     server.ArgString(args, "location"),
			"ssh_key_ids":  server.ArgStringArray(args, "ssh_key_ids"),
		}
		setBody(body, "description", server.ArgString(args, "description"))
		raw, err := gpuRequest("POST", "/deployments/clusters", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_action",
		Description: "Run an action on a GPU deployment (e.g. start, stop, reboot).",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":     server.StringProp("Deployment ID"),
			"action": server.StringProp("Action to perform"),
		}, []string{"id", "action"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		action := server.ArgString(args, "action")
		if id == "" || action == "" {
			return nil, fmt.Errorf("id and action are required")
		}
		raw, err := gpuRequest("POST", "/deployments/"+id+"/actions", nil, map[string]any{"action": action}, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_delete",
		Description: "Delete a GPU deployment.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Deployment ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("DELETE", "/deployments/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_check",
		Description: "Check availability of a specific GPU cluster type.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"cluster_type": server.StringProp("Cluster type to check"),
		}, []string{"cluster_type"}),
	}, func(args map[string]any) (any, error) {
		ct := server.ArgString(args, "cluster_type")
		if ct == "" {
			return nil, fmt.Errorf("cluster_type is required")
		}
		raw, err := gpuRequest("GET", "/deployments/availability/"+ct, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})
}

func registerGPUWaitlistTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_waitlist_list",
		Description: "List GPU waitlist requests.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/deployments/requests", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["data"], nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_waitlist_add",
		Description: "Create a GPU waitlist request. Optionally configure auto-deploy when capacity becomes available.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"cluster_type": server.StringProp("Cluster type to wait for"),
			"location":     server.StringProp("Preferred location (optional)"),
			"auto_deploy":  server.BoolProp("Auto-deploy when available (default false)"),
			"image":        server.StringProp("Auto-deploy image (if auto_deploy is true)"),
			"hostname":     server.StringProp("Auto-deploy hostname (if auto_deploy is true)"),
			"ssh_key_ids":  server.StringArrayProp("Auto-deploy SSH key IDs (if auto_deploy is true)"),
		}, []string{"cluster_type"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"cluster_type": server.ArgString(args, "cluster_type"),
			"auto_deploy":  server.ArgBool(args, "auto_deploy", false),
		}
		setBody(body, "location", server.ArgString(args, "location"))
		if server.ArgBool(args, "auto_deploy", false) {
			cfg := map[string]any{}
			setBody(cfg, "image", server.ArgString(args, "image"))
			setBody(cfg, "hostname", server.ArgString(args, "hostname"))
			setBodyStringArray(cfg, "ssh_key_ids", server.ArgStringArray(args, "ssh_key_ids"))
			if len(cfg) > 0 {
				body["config"] = cfg
			}
		}
		raw, err := gpuRequest("POST", "/deployments/requests", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_waitlist_cancel",
		Description: "Cancel a GPU waitlist request.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Waitlist request ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("DELETE", "/deployments/requests/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerGPUImageTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_image_list",
		Description: "List available GPU images. Optionally filter by cluster type or image type.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"cluster_type": server.StringProp("Filter by compatible cluster type"),
			"image_type":   server.StringProp("Image type filter"),
		}, nil),
	}, func(args map[string]any) (any, error) {
		q := map[string]string{}
		setQuery(q, "cluster_type", server.ArgString(args, "cluster_type"))
		setQuery(q, "image_type", server.ArgString(args, "image_type"))
		raw, err := gpuRequest("GET", "/images", q, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractGPUListWithMeta(raw), nil
	})
}

func registerGPUVolumeTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_volume_list",
		Description: "List GPU volumes.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"status": server.StringProp("Filter by volume status"),
		}, nil),
	}, func(args map[string]any) (any, error) {
		q := map[string]string{}
		setQuery(q, "status", server.ArgString(args, "status"))
		raw, err := gpuRequest("GET", "/volumes", q, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractGPUListWithMeta(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_volume_create",
		Description: "Create a GPU volume.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":         server.StringProp("Volume name"),
			"type":         server.StringProp("Volume type"),
			"location":     server.StringProp("Location code"),
			"size":         server.IntProp("Size in GB"),
			"instance_id":  server.StringProp("Instance ID to attach to (optional)"),
			"instance_ids": server.StringArrayProp("Instance IDs for shared volume (optional)"),
		}, []string{"name", "type", "location", "size"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"name":     server.ArgString(args, "name"),
			"type":     server.ArgString(args, "type"),
			"location": server.ArgString(args, "location"),
			"size":     server.ArgInt(args, "size"),
		}
		setBody(body, "instance_id", server.ArgString(args, "instance_id"))
		setBodyStringArray(body, "instance_ids", server.ArgStringArray(args, "instance_ids"))
		raw, err := gpuRequest("POST", "/volumes", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_volume_delete",
		Description: "Delete a GPU volume.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Volume ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("DELETE", "/volumes/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerGPUSSHKeyTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_ssh_key_list",
		Description: "List GPU SSH keys.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/ssh-keys", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["data"], nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_ssh_key_upload",
		Description: "Upload a GPU SSH key.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":       server.StringProp("Key name"),
			"public_key": server.StringProp("SSH public key content"),
		}, []string{"name", "public_key"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"name":       server.ArgString(args, "name"),
			"public_key": server.ArgString(args, "public_key"),
		}
		raw, err := gpuRequest("POST", "/ssh-keys", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_ssh_key_delete",
		Description: "Delete a GPU SSH key.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("SSH key ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("DELETE", "/ssh-keys/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerGPUAPIKeyTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_api_key_list",
		Description: "List GPU API keys.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/api-keys", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["data"], nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_api_key_create",
		Description: "Create a GPU API key.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name": server.StringProp("API key name"),
		}, []string{"name"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{"name": server.ArgString(args, "name")}
		raw, err := gpuRequest("POST", "/api-keys", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_api_key_revoke",
		Description: "Revoke a GPU API key.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("API key ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("DELETE", "/api-keys/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})
}

func registerGPUWebhookTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_webhook_list",
		Description: "List GPU webhooks.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/webhooks", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["data"], nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_webhook_create",
		Description: "Create a GPU webhook.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"url":    server.StringProp("Webhook URL"),
			"events": server.StringArrayProp("Event names to subscribe to"),
		}, []string{"url", "events"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"url":    server.ArgString(args, "url"),
			"events": server.ArgStringArray(args, "events"),
		}
		raw, err := gpuRequest("POST", "/webhooks", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_webhook_update",
		Description: "Update a GPU webhook.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":        server.StringProp("Webhook ID"),
			"url":       server.StringProp("Webhook URL (optional)"),
			"events":    server.StringArrayProp("Event names (optional)"),
			"is_active": server.BoolProp("Set webhook active state"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		body := map[string]any{"is_active": server.ArgBool(args, "is_active", true)}
		setBody(body, "url", server.ArgString(args, "url"))
		setBodyStringArray(body, "events", server.ArgStringArray(args, "events"))
		raw, err := gpuRequest("PUT", "/webhooks/"+id, nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractData(raw), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_webhook_delete",
		Description: "Delete a GPU webhook.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Webhook ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := gpuRequest("DELETE", "/webhooks/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerGPURegionTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_region_list",
		Description: "List available GPU regions/locations.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/regions", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["data"], nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_gpu_volume_type_list",
		Description: "List GPU volume types.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := gpuRequest("GET", "/volume-types", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["data"], nil
	})
}
