package cmd

import (
	"context"
	"fmt"

	"github.com/kubeopencode/kubeopencode/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	podsCmd := newPodsCmd()
	rootCmd.AddCommand(podsCmd)
}

func newPodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pods",
		Short: "List pods in the current namespace",
		RunE:  runPods,
	}
	return cmd
}

func runPods(cmd *cobra.Command, _ []string) error {
	kubeconfigPath, err := cmd.Root().PersistentFlags().GetString("kubeconfig")
	if err != nil {
		return fmt.Errorf("reading kubeconfig flag: %w", err)
	}

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil {
		return fmt.Errorf("reading namespace flag: %w", err)
	}

	client, err := k8s.NewClientBuilder(kubeconfigPath, namespace).Build()
	if err != nil {
		return fmt.Errorf("building kubernetes client: %w", err)
	}

	lister := k8s.NewPodLister(client)
	pods, err := lister.List(context.Background())
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}

	if len(pods) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No pods found.")
		return nil
	}

	for _, pod := range pods {
		fmt.Fprintf(cmd.OutOrStdout(), "%-40s %s\n", pod.Name, string(pod.Status.Phase))
	}
	return nil
}
