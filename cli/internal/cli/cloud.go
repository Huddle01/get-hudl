package cli

import (
	"fmt"

	"github.com/Huddle01/get-hudl/internal/runtime"
	"github.com/spf13/cobra"
)

func newVMCommand() *cobra.Command {
	vm := &cobra.Command{Use: "vm", Short: "Manage virtual machines"}
	vm.AddCommand(newVMListCommand(), newVMGetCommand(), newVMCreateCommand(), newVMDeleteCommand(), newVMStatusCommand(), newVMActionCommand(), newVMAttachNetworkCommand())
	return vm
}

func newVMListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List virtual machines",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud,
				Method:  "GET",
				Path:    "/instances",
				Query:   map[string]string{"region": region},
			}, extractCloudList("instances"), runtime.MutateOptions{})
		},
	}
}

func newVMGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get VM details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud,
				Method:  "GET",
				Path:    "/instances/" + args[0],
				Query:   map[string]string{"region": region},
			}, extractByKey("instance"), runtime.MutateOptions{})
		},
	}
	cmd.ValidArgsFunction = completeCloudResource("/instances", "instances")
	return cmd
}

func newVMCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var flavorID string
	var imageID string
	var bootDisk int
	var addVolume int
	var keys []string
	var sgs []string
	var tags []string
	var assignIP bool = true

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a VM",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}

			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			body = runtime.MergeRequest(cloneRequest(app.Config.Defaults.VM), body)
			if len(args) > 0 {
				body["name"] = args[0]
			}
			setMapString(body, "flavor_id", flavorID)
			setMapString(body, "image_id", imageID)
			setMapInt(body, "boot_disk_size", bootDisk)
			setMapInt(body, "additional_volume_size", addVolume)
			setMapStringArray(body, "key_name", keys)
			setMapStringArray(body, "sg_names", sgs)
			setMapStringArray(body, "tags", tags)
			body["assign_public_ip"] = assignIP

			if mut.Interactive && app.IsTTYIn {
				if body["name"] == nil {
					value, err := runtime.PromptString(app.Stdin, app.Stderr, "VM name", "", true)
					if err != nil {
						return renderError(app, err)
					}
					body["name"] = value
				}
				if asString(body["flavor_id"]) == "" {
					value, err := runtime.PromptString(app.Stdin, app.Stderr, "Flavor ID", "", true)
					if err != nil {
						return renderError(app, err)
					}
					body["flavor_id"] = value
				}
				if asString(body["image_id"]) == "" {
					value, err := runtime.PromptString(app.Stdin, app.Stderr, "Image ID", "", true)
					if err != nil {
						return renderError(app, err)
					}
					body["image_id"] = value
				}
				if _, ok := body["boot_disk_size"]; !ok {
					value, err := runtime.PromptString(app.Stdin, app.Stderr, "Boot disk size (GB)", "50", true)
					if err != nil {
						return renderError(app, err)
					}
					number, err := parsePositiveInt(value, "boot disk size")
					if err != nil {
						return renderError(app, err)
					}
					body["boot_disk_size"] = number
				}
				if len(stringList(body["key_name"])) == 0 {
					value, err := runtime.PromptCSV(app.Stdin, app.Stderr, "SSH key names (comma-separated)", nil, true)
					if err != nil {
						return renderError(app, err)
					}
					body["key_name"] = value
				}
				if len(stringList(body["sg_names"])) == 0 {
					value, err := runtime.PromptCSV(app.Stdin, app.Stderr, "Security group names (comma-separated)", nil, true)
					if err != nil {
						return renderError(app, err)
					}
					body["sg_names"] = value
				}
			}

			for _, required := range []string{"name", "flavor_id", "image_id", "boot_disk_size", "key_name", "sg_names"} {
				if !hasValue(body[required]) {
					return renderError(app, fmt.Errorf("%s is required", required))
				}
			}

			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendCloud,
				Method:         "POST",
				Path:           "/instances",
				Query:          map[string]string{"region": region},
				Body:           body,
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&flavorID, "flavor", "", "Flavor ID")
	cmd.Flags().StringVar(&imageID, "image", "", "Image ID")
	cmd.Flags().IntVar(&bootDisk, "boot-disk-size", 0, "Boot disk size in GB")
	cmd.Flags().IntVar(&addVolume, "additional-volume-size", 0, "Additional volume size in GB")
	cmd.Flags().StringArrayVar(&keys, "key", nil, "SSH key name (repeatable)")
	cmd.Flags().StringArrayVar(&sgs, "sg", nil, "Security group name (repeatable)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable)")
	cmd.Flags().BoolVar(&assignIP, "assign-public-ip", true, "Assign a public IP")
	return cmd
}

func newVMDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			if err := ensureConfirmation(app, mut, "Delete VM "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendCloud,
				Method:         "DELETE",
				Path:           "/instances/" + args[0],
				Query:          map[string]string{"region": region},
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	cmd.ValidArgsFunction = completeCloudResource("/instances", "instances")
	return cmd
}

func newVMStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Get VM status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud,
				Method:  "GET",
				Path:    "/instances/" + args[0] + "/status",
				Query:   map[string]string{"region": region},
			}, extractByKey(""), runtime.MutateOptions{})
		},
	}
	cmd.ValidArgsFunction = completeCloudResource("/instances", "instances")
	return cmd
}

func newVMActionCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var actionType string
	cmd := &cobra.Command{
		Use:   "action <id> <action>",
		Short: "Run a lifecycle action on a VM",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			body := map[string]any{"action": args[1]}
			setMapString(body, "type", actionType)
			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendCloud,
				Method:         "POST",
				Path:           "/instances/" + args[0] + "/action",
				Query:          map[string]string{"region": region},
				Body:           body,
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	cmd.Flags().StringVar(&actionType, "type", "", "Reboot type: SOFT or HARD")
	return cmd
}

func newVMAttachNetworkCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var networkID string
	var subnetID string
	var fixedIP string
	cmd := &cobra.Command{
		Use:   "attach-network <id>",
		Short: "Attach a network to a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			if networkID == "" {
				return renderError(app, fmt.Errorf("network ID is required"))
			}
			body := map[string]any{"network_id": networkID}
			setMapString(body, "subnet_id", subnetID)
			setMapString(body, "fixed_ip", fixedIP)
			return handleRequest(app, runtime.Request{
				Backend:        runtime.BackendCloud,
				Method:         "POST",
				Path:           "/instances/" + args[0] + "/networks",
				Query:          map[string]string{"region": region},
				Body:           body,
				Mutating:       true,
				IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	cmd.Flags().StringVar(&networkID, "network-id", "", "Network ID to attach")
	cmd.Flags().StringVar(&subnetID, "subnet-id", "", "Optional subnet ID")
	cmd.Flags().StringVar(&fixedIP, "fixed-ip", "", "Optional fixed IP")
	return cmd
}

func newVolumeCommand() *cobra.Command {
	volume := &cobra.Command{Use: "volume", Short: "Manage block storage volumes"}
	volume.AddCommand(newVolumeListCommand(), newVolumeGetCommand(), newVolumeCreateCommand(), newVolumeDeleteCommand(), newVolumeAttachCommand(), newVolumeDetachCommand())
	return volume
}

func newVolumeListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List block volumes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud, Method: "GET", Path: "/volumes", Query: map[string]string{"region": region},
			}, extractCloudList("volumes"), runtime.MutateOptions{})
		},
	}
}

func newVolumeGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud, Method: "GET", Path: "/volumes/" + args[0], Query: map[string]string{"region": region},
			}, extractByKey("volume"), runtime.MutateOptions{})
		},
	}
}

func newVolumeCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var size int
	var description string
	var volumeType string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a block volume",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			body, err := mustLoadRequest(mut)
			if err != nil {
				return renderError(app, err)
			}
			body = runtime.MergeRequest(cloneRequest(app.Config.Defaults.Volume), body)
			if len(args) > 0 {
				body["name"] = args[0]
			}
			setMapInt(body, "size", size)
			setMapString(body, "description", description)
			setMapString(body, "volume_type", volumeType)
			if !hasValue(body["name"]) || !hasValue(body["size"]) {
				return renderError(app, fmt.Errorf("name and size are required"))
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud, Method: "POST", Path: "/volumes", Query: map[string]string{"region": region}, Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().IntVar(&size, "size", 0, "Volume size in GB")
	cmd.Flags().StringVar(&description, "description", "", "Volume description")
	cmd.Flags().StringVar(&volumeType, "type", "", "Volume type")
	return cmd
}

func newVolumeDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			if err := ensureConfirmation(app, mut, "Delete volume "+args[0]); err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud, Method: "DELETE", Path: "/volumes/" + args[0], Query: map[string]string{"region": region}, Mutating: true, IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newVolumeAttachCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var instanceID string
	cmd := &cobra.Command{
		Use:   "attach <id>",
		Short: "Attach a volume to a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			if instanceID == "" {
				return renderError(app, fmt.Errorf("instance ID is required"))
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud, Method: "POST", Path: "/volumes/" + args[0] + "/attach", Query: map[string]string{"region": region}, Body: map[string]any{"instance_id": instanceID}, Mutating: true, IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	cmd.Flags().StringVar(&instanceID, "to", "", "Instance ID to attach to")
	return cmd
}

func newVolumeDetachCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var instanceID string
	cmd := &cobra.Command{
		Use:   "detach <id>",
		Short: "Detach a volume from a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			if instanceID == "" {
				return renderError(app, fmt.Errorf("instance ID is required"))
			}
			return handleRequest(app, runtime.Request{
				Backend: runtime.BackendCloud, Method: "POST", Path: "/volumes/" + args[0] + "/detach", Query: map[string]string{"region": region}, Body: map[string]any{"instance_id": instanceID}, Mutating: true, IdempotencyKey: mut.IdempotencyKey,
			}, extractByKey(""), mut)
		},
	}
	addMutateFlags(cmd, &mut, false)
	cmd.Flags().StringVar(&instanceID, "from", "", "Instance ID to detach from")
	return cmd
}

func newFloatingIPCommand() *cobra.Command {
	fip := &cobra.Command{Use: "fip", Short: "Manage floating IPs"}
	fip.AddCommand(newFIPListCommand(), newFIPGetCommand(), newFIPAssociateCommand(), newFIPDisassociateCommand())
	return fip
}

func newFIPListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List floating IPs", RunE: func(cmd *cobra.Command, _ []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/floating-ips", Query: map[string]string{"region": region}}, extractCloudList("floating_ips"), runtime.MutateOptions{})
	}}
}

func newFIPGetCommand() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get a floating IP", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/floating-ips/" + args[0], Query: map[string]string{"region": region}}, extractByKey("floating_ip"), runtime.MutateOptions{})
	}}
}

func newFIPAssociateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var instanceID string
	cmd := &cobra.Command{Use: "associate <id>", Short: "Associate a floating IP", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		if instanceID == "" {
			return renderError(app, fmt.Errorf("instance ID is required"))
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/floating-ips/" + args[0] + "/associate", Query: map[string]string{"region": region}, Body: map[string]any{"instance_id": instanceID}, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	cmd.Flags().StringVar(&instanceID, "to", "", "Instance ID to associate")
	return cmd
}

func newFIPDisassociateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{Use: "disassociate <id>", Short: "Disassociate a floating IP", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/floating-ips/" + args[0] + "/disassociate", Query: map[string]string{"region": region}, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newSecurityGroupCommand() *cobra.Command {
	sg := &cobra.Command{Use: "sg", Short: "Manage security groups"}
	sg.AddCommand(newSGListCommand(), newSGGetCommand(), newSGCreateCommand(), newSGDeleteCommand(), newSGDuplicateCommand(), newSGRuleCommand())
	return sg
}

func newSGListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List security groups", RunE: func(cmd *cobra.Command, _ []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/security-groups", Query: map[string]string{"region": region}}, extractCloudList("security_groups"), runtime.MutateOptions{})
	}}
}

func newSGGetCommand() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get a security group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/security-groups/" + args[0], Query: map[string]string{"region": region}}, extractByKey("security_group"), runtime.MutateOptions{})
	}}
}

func newSGCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var description string
	cmd := &cobra.Command{Use: "create <name>", Short: "Create a security group", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		body, err := mustLoadRequest(mut)
		if err != nil {
			return renderError(app, err)
		}
		body = runtime.MergeRequest(cloneRequest(app.Config.Defaults.SG), body)
		if len(args) > 0 {
			body["name"] = args[0]
		}
		setMapString(body, "description", description)
		if !hasValue(body["name"]) {
			return renderError(app, fmt.Errorf("name is required"))
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/security-groups", Query: map[string]string{"region": region}, Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&description, "description", "", "Security group description")
	return cmd
}

func newSGDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{Use: "delete <id>", Short: "Delete a security group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		if err := ensureConfirmation(app, mut, "Delete security group "+args[0]); err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "DELETE", Path: "/security-groups/" + args[0], Query: map[string]string{"region": region}, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newSGDuplicateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var targetRegion string
	var name string
	cmd := &cobra.Command{Use: "duplicate <id>", Short: "Duplicate a security group into another region", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		if targetRegion == "" {
			return renderError(app, fmt.Errorf("target region is required"))
		}
		body := map[string]any{"target_region": targetRegion}
		setMapString(body, "name", name)
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/security-groups/" + args[0] + "/duplicate", Query: map[string]string{"region": region}, Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	cmd.Flags().StringVar(&targetRegion, "target-region", "", "Region to duplicate into")
	cmd.Flags().StringVar(&name, "name", "", "Optional new name")
	return cmd
}

func newSGRuleCommand() *cobra.Command {
	rule := &cobra.Command{Use: "rule", Short: "Manage security group rules"}
	rule.AddCommand(newSGRuleAddCommand(), newSGRuleDeleteCommand())
	return rule
}

func newSGRuleAddCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var direction string
	var etherType string
	var protocol string
	var portMin int
	var portMax int
	var remoteIP string
	var remoteGroup string
	cmd := &cobra.Command{Use: "add <sg-id>", Short: "Add a rule to a security group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		body, err := mustLoadRequest(mut)
		if err != nil {
			return renderError(app, err)
		}
		setMapString(body, "direction", direction)
		setMapString(body, "ether_type", etherType)
		setMapString(body, "protocol", protocol)
		setMapInt(body, "port_range_min", portMin)
		setMapInt(body, "port_range_max", portMax)
		setMapString(body, "remote_ip_prefix", remoteIP)
		setMapString(body, "remote_group_id", remoteGroup)
		if mut.Interactive && app.IsTTYIn {
			if asString(body["direction"]) == "" {
				value, err := runtime.PromptString(app.Stdin, app.Stderr, "Direction (ingress/egress)", "", true)
				if err != nil {
					return renderError(app, err)
				}
				body["direction"] = value
			}
			if asString(body["ether_type"]) == "" {
				value, err := runtime.PromptString(app.Stdin, app.Stderr, "Ether type (IPv4/IPv6)", "", true)
				if err != nil {
					return renderError(app, err)
				}
				body["ether_type"] = value
			}
		}
		if !hasValue(body["direction"]) || !hasValue(body["ether_type"]) {
			return renderError(app, fmt.Errorf("direction and ether_type are required"))
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/security-groups/" + args[0] + "/rules", Query: map[string]string{"region": region}, Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&direction, "direction", "", "Traffic direction: ingress or egress")
	cmd.Flags().StringVar(&etherType, "ether-type", "", "IP family: IPv4 or IPv6")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp")
	cmd.Flags().IntVar(&portMin, "port-min", 0, "Minimum port")
	cmd.Flags().IntVar(&portMax, "port-max", 0, "Maximum port")
	cmd.Flags().StringVar(&remoteIP, "remote-ip-prefix", "", "Remote CIDR")
	cmd.Flags().StringVar(&remoteGroup, "remote-group-id", "", "Remote security group ID")
	return cmd
}

func newSGRuleDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{Use: "delete <sg-id> <rule-id>", Short: "Delete a security group rule", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		if err := ensureConfirmation(app, mut, "Delete security group rule "+args[1]); err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "DELETE", Path: "/security-groups/" + args[0] + "/rules/" + args[1], Query: map[string]string{"region": region}, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newNetworkCommand() *cobra.Command {
	network := &cobra.Command{Use: "network", Short: "Manage private networks"}
	network.AddCommand(newNetworkListCommand(), newNetworkCreateCommand(), newNetworkDeleteCommand())
	return network
}

func newNetworkListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List networks", RunE: func(cmd *cobra.Command, _ []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		extract := func(raw map[string]any) (any, *runtime.Paging, error) {
			if data, ok := raw["data"].(map[string]any); ok {
				return data["networks"], nil, nil
			}
			return []any{}, nil, nil
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/networks", Query: map[string]string{"region": region}}, extract, runtime.MutateOptions{})
	}}
}

func newNetworkCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var description string
	var poolCIDR string
	var primaryCIDR string
	var primarySize int
	var noGateway bool
	var enableDHCP bool = true
	cmd := &cobra.Command{Use: "create <name>", Short: "Create a network", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		body, err := mustLoadRequest(mut)
		if err != nil {
			return renderError(app, err)
		}
		if len(args) > 0 {
			body["name"] = args[0]
		}
		setMapString(body, "description", description)
		setMapString(body, "pool_cidr", poolCIDR)
		setMapString(body, "primary_subnet_cidr", primaryCIDR)
		setMapInt(body, "primary_subnet_size", primarySize)
		body["no_gateway"] = noGateway
		body["enable_dhcp"] = enableDHCP
		if !hasValue(body["name"]) {
			return renderError(app, fmt.Errorf("name is required"))
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/networks", Query: map[string]string{"region": region}, Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&poolCIDR, "pool-cidr", "", "Pool CIDR")
	cmd.Flags().StringVar(&primaryCIDR, "primary-subnet-cidr", "", "Primary subnet CIDR")
	cmd.Flags().IntVar(&primarySize, "primary-subnet-size", 0, "Primary subnet size prefix length")
	cmd.Flags().BoolVar(&noGateway, "no-gateway", false, "Disable gateway on the subnet")
	cmd.Flags().BoolVar(&enableDHCP, "enable-dhcp", true, "Enable DHCP on the subnet")
	return cmd
}

func newNetworkDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{Use: "delete <id>", Short: "Delete a network", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		if err := ensureConfirmation(app, mut, "Delete network "+args[0]); err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "DELETE", Path: "/networks/" + args[0], Query: map[string]string{"region": region}, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newKeyCommand() *cobra.Command {
	key := &cobra.Command{Use: "key", Short: "Manage SSH key pairs"}
	key.AddCommand(newKeyListCommand(), newKeyGetCommand(), newKeyCreateCommand(), newKeyDeleteCommand())
	return key
}

func newKeyListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List key pairs", RunE: func(cmd *cobra.Command, _ []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/keypairs", Query: map[string]string{"region": region}}, extractCloudList("keypairs"), runtime.MutateOptions{})
	}}
}

func newKeyGetCommand() *cobra.Command {
	return &cobra.Command{Use: "get <name>", Short: "Get a key pair", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/keypairs/" + args[0], Query: map[string]string{"region": region}}, extractByKey("keypair"), runtime.MutateOptions{})
	}}
}

func newKeyCreateCommand() *cobra.Command {
	var mut runtime.MutateOptions
	var publicKey string
	cmd := &cobra.Command{Use: "create <name>", Short: "Create a key pair", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
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
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "POST", Path: "/keypairs", Query: map[string]string{"region": region}, Body: body, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, true)
	cmd.Flags().StringVar(&publicKey, "public-key", "", "SSH public key")
	return cmd
}

func newKeyDeleteCommand() *cobra.Command {
	var mut runtime.MutateOptions
	cmd := &cobra.Command{Use: "delete <name>", Short: "Delete a key pair", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := appFromCommand(cmd)
		region, err := regionFromApp(app, true)
		if err != nil {
			return renderError(app, err)
		}
		if err := ensureConfirmation(app, mut, "Delete key pair "+args[0]); err != nil {
			return renderError(app, err)
		}
		return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "DELETE", Path: "/keypairs/" + args[0], Query: map[string]string{"region": region}, Mutating: true, IdempotencyKey: mut.IdempotencyKey}, extractByKey(""), mut)
	}}
	addMutateFlags(cmd, &mut, false)
	return cmd
}

func newFlavorCommand() *cobra.Command {
	flavor := &cobra.Command{Use: "flavor", Short: "Discover compute flavors"}
	flavor.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List flavors",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/flavors", Query: map[string]string{"region": region}}, extractCloudList("flavors"), runtime.MutateOptions{})
		},
	})
	return flavor
}

func newImageCommand() *cobra.Command {
	image := &cobra.Command{Use: "image", Short: "Discover available VM images"}
	image.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List VM images",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			region, err := regionFromApp(app, true)
			if err != nil {
				return renderError(app, err)
			}
			extract := func(raw map[string]any) (any, *runtime.Paging, error) {
				return normalizeCloudImageGroups(raw["image_groups"]), nil, nil
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/images", Query: map[string]string{"region": region}}, extract, runtime.MutateOptions{})
		},
	})
	return image
}

func newRegionCommand() *cobra.Command {
	region := &cobra.Command{Use: "region", Short: "Discover available regions"}
	region.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List regions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCommand(cmd)
			extract := func(raw map[string]any) (any, *runtime.Paging, error) {
				rows, err := normalizeRegions(raw)
				return rows, nil, err
			}
			return handleRequest(app, runtime.Request{Backend: runtime.BackendCloud, Method: "GET", Path: "/regions"}, extract, runtime.MutateOptions{})
		},
	})
	return region
}
