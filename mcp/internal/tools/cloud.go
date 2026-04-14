package tools

import (
	"fmt"

	"github.com/Huddle01/get-hudl/internal/runtime"
	"github.com/Huddle01/get-hudl/mcp/internal/server"
)

// cloudDo is a region-free cloud request for global endpoints (flavors, images, regions).
func cloudDo(path string) (map[string]any, error) {
	return do(runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: path})
}

func registerVMTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_list",
		Description: "List all virtual machines in the current region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/instances", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "instances"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_get",
		Description: "Get details of a virtual machine by ID.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("VM instance ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("GET", "/instances/"+id, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractKey(raw, "instance"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_create",
		Description: "Create a new virtual machine. Requires name, flavor_id, image_id, boot_disk_size, key_name, and sg_names.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":                   server.StringProp("VM name"),
			"flavor_id":              server.StringProp("Flavor ID (compute size)"),
			"image_id":               server.StringProp("OS image ID"),
			"boot_disk_size":         server.IntProp("Boot disk size in GB"),
			"additional_volume_size": server.IntProp("Additional volume size in GB (optional)"),
			"key_name":               server.StringArrayProp("SSH key names"),
			"sg_names":               server.StringArrayProp("Security group names"),
			"tags":                   server.StringArrayProp("Tags (optional)"),
			"assign_public_ip":       server.BoolProp("Assign a public IP (default true)"),
		}, []string{"name", "flavor_id", "image_id", "boot_disk_size", "key_name", "sg_names"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"name":             server.ArgString(args, "name"),
			"flavor_id":       server.ArgString(args, "flavor_id"),
			"image_id":        server.ArgString(args, "image_id"),
			"boot_disk_size":  server.ArgInt(args, "boot_disk_size"),
			"key_name":        server.ArgStringArray(args, "key_name"),
			"sg_names":        server.ArgStringArray(args, "sg_names"),
			"assign_public_ip": server.ArgBool(args, "assign_public_ip", true),
		}
		setBodyInt(body, "additional_volume_size", server.ArgInt(args, "additional_volume_size"))
		setBodyStringArray(body, "tags", server.ArgStringArray(args, "tags"))
		raw, err := cloudRequest("POST", "/instances", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_delete",
		Description: "Delete a virtual machine by ID.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("VM instance ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("DELETE", "/instances/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_status",
		Description: "Get the current status of a virtual machine.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("VM instance ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("GET", "/instances/"+id+"/status", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_action",
		Description: "Run a lifecycle action on a VM (e.g. start, stop, reboot, pause, resume, rebuild).",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":     server.StringProp("VM instance ID"),
			"action": server.StringProp("Action to perform (start, stop, reboot, pause, resume, rebuild)"),
			"type":   server.StringProp("Reboot type: SOFT or HARD (only for reboot action)"),
		}, []string{"id", "action"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		action := server.ArgString(args, "action")
		if id == "" || action == "" {
			return nil, fmt.Errorf("id and action are required")
		}
		body := map[string]any{"action": action}
		setBody(body, "type", server.ArgString(args, "type"))
		raw, err := cloudRequest("POST", "/instances/"+id+"/action", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_vm_attach_network",
		Description: "Attach a private network to a VM.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":         server.StringProp("VM instance ID"),
			"network_id": server.StringProp("Network ID to attach"),
			"subnet_id":  server.StringProp("Optional subnet ID"),
			"fixed_ip":   server.StringProp("Optional fixed IP address"),
		}, []string{"id", "network_id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		networkID := server.ArgString(args, "network_id")
		if id == "" || networkID == "" {
			return nil, fmt.Errorf("id and network_id are required")
		}
		body := map[string]any{"network_id": networkID}
		setBody(body, "subnet_id", server.ArgString(args, "subnet_id"))
		setBody(body, "fixed_ip", server.ArgString(args, "fixed_ip"))
		raw, err := cloudRequest("POST", "/instances/"+id+"/networks", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerVolumeTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_volume_list",
		Description: "List all block storage volumes in the current region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/volumes", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "volumes"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_volume_get",
		Description: "Get details of a block storage volume.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Volume ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("GET", "/volumes/"+id, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractKey(raw, "volume"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_volume_create",
		Description: "Create a block storage volume.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":        server.StringProp("Volume name"),
			"size":        server.IntProp("Size in GB"),
			"description": server.StringProp("Volume description (optional)"),
			"volume_type": server.StringProp("Volume type (optional)"),
		}, []string{"name", "size"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"name": server.ArgString(args, "name"),
			"size": server.ArgInt(args, "size"),
		}
		setBody(body, "description", server.ArgString(args, "description"))
		setBody(body, "volume_type", server.ArgString(args, "volume_type"))
		raw, err := cloudRequest("POST", "/volumes", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_volume_delete",
		Description: "Delete a block storage volume.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Volume ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("DELETE", "/volumes/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_volume_attach",
		Description: "Attach a volume to a VM instance.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":          server.StringProp("Volume ID"),
			"instance_id": server.StringProp("Instance ID to attach to"),
		}, []string{"id", "instance_id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		instanceID := server.ArgString(args, "instance_id")
		if id == "" || instanceID == "" {
			return nil, fmt.Errorf("id and instance_id are required")
		}
		raw, err := cloudRequest("POST", "/volumes/"+id+"/attach", nil, map[string]any{"instance_id": instanceID}, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_volume_detach",
		Description: "Detach a volume from a VM instance.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":          server.StringProp("Volume ID"),
			"instance_id": server.StringProp("Instance ID to detach from"),
		}, []string{"id", "instance_id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		instanceID := server.ArgString(args, "instance_id")
		if id == "" || instanceID == "" {
			return nil, fmt.Errorf("id and instance_id are required")
		}
		raw, err := cloudRequest("POST", "/volumes/"+id+"/detach", nil, map[string]any{"instance_id": instanceID}, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerFloatingIPTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_fip_list",
		Description: "List all floating IPs in the current region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/floating-ips", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "floating_ips"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_fip_get",
		Description: "Get details of a floating IP.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Floating IP ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("GET", "/floating-ips/"+id, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractKey(raw, "floating_ip"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_fip_associate",
		Description: "Associate a floating IP with a VM instance.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":          server.StringProp("Floating IP ID"),
			"instance_id": server.StringProp("Instance ID to associate with"),
		}, []string{"id", "instance_id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		instanceID := server.ArgString(args, "instance_id")
		if id == "" || instanceID == "" {
			return nil, fmt.Errorf("id and instance_id are required")
		}
		raw, err := cloudRequest("POST", "/floating-ips/"+id+"/associate", nil, map[string]any{"instance_id": instanceID}, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_fip_disassociate",
		Description: "Disassociate a floating IP from its current instance.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Floating IP ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("POST", "/floating-ips/"+id+"/disassociate", nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerSecurityGroupTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_list",
		Description: "List all security groups in the current region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/security-groups", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "security_groups"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_get",
		Description: "Get details of a security group, including its rules.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Security group ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("GET", "/security-groups/"+id, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractKey(raw, "security_group"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_create",
		Description: "Create a security group.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":        server.StringProp("Security group name"),
			"description": server.StringProp("Description (optional)"),
		}, []string{"name"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{"name": server.ArgString(args, "name")}
		setBody(body, "description", server.ArgString(args, "description"))
		raw, err := cloudRequest("POST", "/security-groups", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_delete",
		Description: "Delete a security group.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Security group ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("DELETE", "/security-groups/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_duplicate",
		Description: "Duplicate a security group into another region.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id":            server.StringProp("Security group ID to duplicate"),
			"target_region": server.StringProp("Target region code"),
			"name":          server.StringProp("Optional new name for the duplicate"),
		}, []string{"id", "target_region"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		targetRegion := server.ArgString(args, "target_region")
		if id == "" || targetRegion == "" {
			return nil, fmt.Errorf("id and target_region are required")
		}
		body := map[string]any{"target_region": targetRegion}
		setBody(body, "name", server.ArgString(args, "name"))
		raw, err := cloudRequest("POST", "/security-groups/"+id+"/duplicate", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_rule_add",
		Description: "Add a firewall rule to a security group.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"sg_id":            server.StringProp("Security group ID"),
			"direction":        server.EnumProp("Traffic direction", []string{"ingress", "egress"}),
			"ether_type":       server.EnumProp("IP family", []string{"IPv4", "IPv6"}),
			"protocol":         server.StringProp("Protocol: tcp, udp, icmp (optional)"),
			"port_range_min":   server.IntProp("Minimum port (optional)"),
			"port_range_max":   server.IntProp("Maximum port (optional)"),
			"remote_ip_prefix": server.StringProp("Remote CIDR block (optional)"),
			"remote_group_id":  server.StringProp("Remote security group ID (optional)"),
		}, []string{"sg_id", "direction", "ether_type"}),
	}, func(args map[string]any) (any, error) {
		sgID := server.ArgString(args, "sg_id")
		if sgID == "" {
			return nil, fmt.Errorf("sg_id is required")
		}
		body := map[string]any{
			"direction":  server.ArgString(args, "direction"),
			"ether_type": server.ArgString(args, "ether_type"),
		}
		setBody(body, "protocol", server.ArgString(args, "protocol"))
		setBodyInt(body, "port_range_min", server.ArgInt(args, "port_range_min"))
		setBodyInt(body, "port_range_max", server.ArgInt(args, "port_range_max"))
		setBody(body, "remote_ip_prefix", server.ArgString(args, "remote_ip_prefix"))
		setBody(body, "remote_group_id", server.ArgString(args, "remote_group_id"))
		raw, err := cloudRequest("POST", "/security-groups/"+sgID+"/rules", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_sg_rule_delete",
		Description: "Delete a firewall rule from a security group.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"sg_id":   server.StringProp("Security group ID"),
			"rule_id": server.StringProp("Rule ID to delete"),
		}, []string{"sg_id", "rule_id"}),
	}, func(args map[string]any) (any, error) {
		sgID := server.ArgString(args, "sg_id")
		ruleID := server.ArgString(args, "rule_id")
		if sgID == "" || ruleID == "" {
			return nil, fmt.Errorf("sg_id and rule_id are required")
		}
		raw, err := cloudRequest("DELETE", "/security-groups/"+sgID+"/rules/"+ruleID, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerNetworkTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_network_list",
		Description: "List all private networks in the current region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/networks", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "networks"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_network_create",
		Description: "Create a private network.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":                server.StringProp("Network name"),
			"description":        server.StringProp("Description (optional)"),
			"pool_cidr":          server.StringProp("Pool CIDR (optional)"),
			"primary_subnet_cidr": server.StringProp("Primary subnet CIDR (optional)"),
			"primary_subnet_size": server.IntProp("Primary subnet prefix length (optional)"),
			"no_gateway":         server.BoolProp("Disable gateway (default false)"),
			"enable_dhcp":        server.BoolProp("Enable DHCP (default true)"),
		}, []string{"name"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"name":        server.ArgString(args, "name"),
			"no_gateway":  server.ArgBool(args, "no_gateway", false),
			"enable_dhcp": server.ArgBool(args, "enable_dhcp", true),
		}
		setBody(body, "description", server.ArgString(args, "description"))
		setBody(body, "pool_cidr", server.ArgString(args, "pool_cidr"))
		setBody(body, "primary_subnet_cidr", server.ArgString(args, "primary_subnet_cidr"))
		setBodyInt(body, "primary_subnet_size", server.ArgInt(args, "primary_subnet_size"))
		raw, err := cloudRequest("POST", "/networks", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_network_delete",
		Description: "Delete a private network.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"id": server.StringProp("Network ID"),
		}, []string{"id"}),
	}, func(args map[string]any) (any, error) {
		id := server.ArgString(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		raw, err := cloudRequest("DELETE", "/networks/"+id, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerKeyTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_key_list",
		Description: "List SSH key pairs in the current region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/keypairs", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "keypairs"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_key_get",
		Description: "Get details of an SSH key pair.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name": server.StringProp("Key pair name"),
		}, []string{"name"}),
	}, func(args map[string]any) (any, error) {
		name := server.ArgString(args, "name")
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		raw, err := cloudRequest("GET", "/keypairs/"+name, nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractKey(raw, "keypair"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_key_create",
		Description: "Create an SSH key pair by uploading a public key.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name":       server.StringProp("Key pair name"),
			"public_key": server.StringProp("SSH public key content"),
		}, []string{"name", "public_key"}),
	}, func(args map[string]any) (any, error) {
		body := map[string]any{
			"name":       server.ArgString(args, "name"),
			"public_key": server.ArgString(args, "public_key"),
		}
		raw, err := cloudRequest("POST", "/keypairs", nil, body, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_key_delete",
		Description: "Delete an SSH key pair.",
		InputSchema: server.ObjectSchema("", map[string]any{
			"name": server.StringProp("Key pair name"),
		}, []string{"name"}),
	}, func(args map[string]any) (any, error) {
		name := server.ArgString(args, "name")
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		raw, err := cloudRequest("DELETE", "/keypairs/"+name, nil, nil, true)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}

func registerLookupTools(srv *server.Server) {
	srv.RegisterTool(server.Tool{
		Name:        "hudl_flavor_list",
		Description: "List available compute flavors (instance sizes and specs). Use this to find valid flavor_id values for hudl_vm_create.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/flavors", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return extractCloudList(raw, "flavors"), nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_image_list",
		Description: "List available VM images (operating systems). Use this to find valid image_id values for hudl_vm_create.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudRequest("GET", "/images", nil, nil, false)
		if err != nil {
			return nil, wrapError(err)
		}
		return raw["image_groups"], nil
	})

	srv.RegisterTool(server.Tool{
		Name:        "hudl_region_list",
		Description: "List available cloud regions. Use this to find valid region codes for hudl_ctx_region.",
		InputSchema: server.ObjectSchema("", map[string]any{}, nil),
	}, func(_ map[string]any) (any, error) {
		raw, err := cloudDo("/regions")
		if err != nil {
			return nil, wrapError(err)
		}
		return raw, nil
	})
}
