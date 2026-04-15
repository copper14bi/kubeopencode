package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kubeopencode/kubeopencode/internal/k8s"
	"github.com/kubeopencode/kubeopencode/internal/kubeconfig"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDeploymentsCmd())
}

func newDeploymentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "deployments",
		Aliases: []string{"deploy", "deployment"},
		Short:   "List deployments in a namespace",
		RunE:    runDeployments,
	}
}

func runDeployments(cmd *cobra.Command, args []string) error {
	kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
	namespace, _ := cmd.Flags().GetString("namespace")

	loader := kubeconfig.NewLoader(kubeconfigPath, namespace)
	ns := loader.Namespace()

	clientBuilder := k8s.NewClientBuilder(kubeconfigPath)
	client, err := clientBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to build k8s client: %w", err)
	}

	lister := k8s.NewDeploymentLister(client, ns)
	deployments, err := lister.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tREADY\tAVAILABLE")
	for _, d := range deployments {
		fmt.Fprintf(w, "%s\t%s\t%d/%d\t%d\n",
			d.Name,
			d.Namespace,
			d.Status.ReadyReplicas,
			d.Status.Replicas,
			d.Status.AvailableReplicas,
		)
	}
	return w.Flush()
}
