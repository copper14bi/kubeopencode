package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kubeopencode/kubeopencode/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newServicesCmd())
}

func newServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "services",
		Aliases: []string{"svc"},
		Short:   "List services in a namespace",
		RunE:    runServices,
	}
	return cmd
}

func runServices(cmd *cobra.Command, args []string) error {
	clientset, err := buildClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to build client: %w", err)
	}

	ns, _ := cmd.Root().PersistentFlags().GetString("namespace")
	lister := k8s.NewServiceLister(clientset, ns)

	svcs, err := lister.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tTYPE\tCLUSTER-IP")
	for _, svc := range svcs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			svc.Name,
			svc.Namespace,
			string(svc.Spec.Type),
			svc.Spec.ClusterIP,
		)
	}
	return w.Flush()
}
