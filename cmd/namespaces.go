package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kubeopencode/kubeopencode/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newNamespacesCmd())
}

func newNamespacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespaces",
		Aliases: []string{"ns"},
		Short:   "List namespaces in the cluster",
		RunE:    runNamespaces,
	}
	return cmd
}

func runNamespaces(cmd *cobra.Command, args []string) error {
	client, err := buildClient(cmd)
	if err != nil {
		return err
	}

	lister := k8s.NewNamespaceLister(client)
	namespaces, err := lister.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("listing namespaces: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS")
	for _, ns := range namespaces {
		fmt.Fprintf(w, "%s\t%s\n", ns.Name, ns.Status.Phase)
	}
	return w.Flush()
}
