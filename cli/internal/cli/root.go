package cli

import (
	"io"

	"github.com/Huddle01/get-hudl/internal/runtime"
	"github.com/spf13/cobra"
)

func NewRootCommand(stdin io.Reader, stdout, stderr io.Writer, version string) *cobra.Command {
	opts := &runtime.GlobalOptions{}
	root := &cobra.Command{
		Use:           "hudl",
		Short:         "Huddle Cloud CLI",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = false

	addGlobalFlags(root, opts)
	root.PersistentPreRunE = newAppPersistentPreRun(stdin, stdout, stderr, opts)
	root.InitDefaultCompletionCmd()
	root.InitDefaultVersionFlag()

	root.AddCommand(
		newLoginCommand(),
		newAuthCommand(),
		newContextCommand(),
		newVMCommand(),
		newVolumeCommand(),
		newFloatingIPCommand(),
		newSecurityGroupCommand(),
		newNetworkCommand(),
		newKeyCommand(),
		newFlavorCommand(),
		newImageCommand(),
		newRegionCommand(),
		newGPUCommand(),
	)

	return root
}
