// Copyright Contributors to the KubeOpenCode project

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeopenv1alpha1 "github.com/kubeopencode/kubeopencode/api/v1alpha1"
)

func init() {
	rootCmd.AddCommand(newGetCmd())
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display KubeOpenCode resources",
	}
	cmd.AddCommand(newGetAgentsCmd())
	cmd.AddCommand(newGetAgentTemplatesCmd())
	cmd.AddCommand(newGetTasksCmd())
	return cmd
}

func newGetAgentsCmd() *cobra.Command {
	var (
		namespace string
		wide      bool
	)

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "List available agents",
		Long: `List agents across all namespaces (or a specific namespace with -n).

Use -o wide to show additional columns (profile, template).

Examples:
  kubeoc get agents
  kubeoc get agents -n production
  kubeoc get agents -o wide`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getKubeConfig()
			if err != nil {
				return fmt.Errorf("cannot connect to cluster: %w", err)
			}

			k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			var agents kubeopenv1alpha1.AgentList
			listOpts := []client.ListOption{}
			if namespace != "" {
				listOpts = append(listOpts, client.InNamespace(namespace))
			}

			if err := k8sClient.List(cmd.Context(), &agents, listOpts...); err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}

			if len(agents.Items) == 0 {
				if namespace != "" {
					fmt.Printf("No agents found in namespace %q\n", namespace)
				} else {
					fmt.Println("No agents found")
				}
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			if wide {
				_, _ = fmt.Fprintln(w, "NAMESPACE\tNAME\tMODE\tSTATUS\tPROFILE\tTEMPLATE")
			} else {
				_, _ = fmt.Fprintln(w, "NAMESPACE\tNAME\tMODE\tSTATUS")
			}

			for _, agent := range agents.Items {
				mode := "Pod"
				status := "-"

				if agent.Spec.ServerConfig != nil {
					mode = "Server"
					switch {
					case agent.Status.ServerStatus != nil && agent.Status.ServerStatus.Suspended:
						status = "Suspended"
					case agent.Status.ServerStatus != nil && agent.Status.ServerStatus.Ready:
						status = "Ready"
					default:
						status = "Not Ready"
					}
				}

				if wide {
					profile := agent.Spec.Profile
					if len(profile) > 50 {
						profile = profile[:47] + "..."
					}

					template := "-"
					if agent.Spec.TemplateRef != nil {
						template = agent.Spec.TemplateRef.Name
					}

					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
						agent.Namespace, agent.Name, mode, status, profile, template)
				} else {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						agent.Namespace, agent.Name, mode, status)
				}
			}

			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Filter by namespace (default: all namespaces)")
	cmd.Flags().BoolVar(&wide, "wide", false, "Show additional columns (profile, template)")

	return cmd
}
