package main

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/stolostron/cluster-templates-operator/cli/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {

	flags := pflag.NewFlagSet("kubectl-cluster", pflag.ExitOnError)
	pflag.CommandLine = flags

	rootCmd := cmd.NewCmdRoot(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
