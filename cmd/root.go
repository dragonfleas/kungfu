package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kungfu",
		Short: "kungfu - patch and extend OpenTofu modules without modifying source",
		Long: `kungfu is a tool to patch and extend the internals of OpenTofu modules
without needing to modify the module source, inspired by Kustomize's approach
to bringing extensibility to declarative configuration.

It takes a source reusable module, applies patches over the top of its internals
in the form of overlays and variants, and generates a new module that can be
utilized by your main OpenTofu code.`,
	}

	cmd.AddCommand(NewBuildCmd())

	return cmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
