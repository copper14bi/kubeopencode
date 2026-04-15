package cmd

import (
	"fmt"

	"github.com/kubeopencode/kubeopencode/internal/kubeconfig"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(kubeconfigCmd)
}

var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Display resolved kubeconfig information",
	Long: `Display information about the resolved kubeconfig path and target namespace.

This command helps verify which kubeconfig file and namespace kubeopencode
will use when connecting to a Kubernetes cluster.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfigPath, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			return fmt.Errorf("failed to read kubeconfig flag: %w", err)
		}
		if kubeconfigPath == "" {
			kubeconfigPath = kubeconfig.DefaultKubeconfigPath()
		}

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			return fmt.Errorf("failed to read namespace flag: %w", err)
		}

		loader := kubeconfig.NewLoader(kubeconfigPath, ns)

		fmt.Fprintf(cmd.OutOrStdout(), "Kubeconfig : %s\n", kubeconfigPath)
		fmt.Fprintf(cmd.OutOrStdout(), "Namespace  : %s\n", loader.Namespace())
		return nil
	},
}
