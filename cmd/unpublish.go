package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1alpha1 "kubesphere.io/api/core/v1alpha1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubesphere/ksbuilder/pkg/utils"
)

type unpublishOptions struct {
	kubeconfig string
}

func defaultUnpublishOptions() *unpublishOptions {
	return &unpublishOptions{}
}

func unpublishExtensionCmd() *cobra.Command {
	o := defaultUnpublishOptions()

	cmd := &cobra.Command{
		Use:   "unpublish",
		Short: "Unpublish an extension from the market",
		Args:  cobra.ExactArgs(1),
		RunE:  o.unpublish,
	}
	cmd.Flags().StringVar(&o.kubeconfig, "kubeconfig", "", "kubeconfig file path of the target cluster")
	return cmd
}

func (o *unpublishOptions) unpublish(_ *cobra.Command, args []string) error {
	name := args[0]
	fmt.Printf("unpublish extension %s\n", name)

	if o.kubeconfig == "" {
		homeDir, _ := os.UserHomeDir()
		o.kubeconfig = fmt.Sprintf("%s/.kube/config", homeDir)
	}
	genericClient, err := utils.BuildClientFromFlags(o.kubeconfig)
	if err != nil {
		return err
	}

	extensionVersions := &corev1alpha1.ExtensionVersionList{}
	if err = genericClient.List(context.Background(), extensionVersions, runtimeclient.MatchingLabels{
		corev1alpha1.ExtensionReferenceLabel: name,
	}); err != nil {
		return err
	}
	objs := make([]runtimeclient.Object, 0)
	for i := range extensionVersions.Items {
		version := &extensionVersions.Items[i]
		objs = append(objs, &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("extension-%s-chart", version.Name),
				Namespace: "kubesphere-system",
			},
		}, version)
	}

	return deleteObjs(genericClient, append(objs, &corev1alpha1.InstallPlan{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubesphere.io/v1alpha1",
			Kind:       "InstallPlan",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, &corev1alpha1.Extension{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubesphere.io/v1alpha1",
			Kind:       "Extension",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})...)
}

func deleteObjs(c runtimeclient.Client, objs ...runtimeclient.Object) error {
	for _, obj := range objs {
		fmt.Printf("deleting %s %s\n", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
		if err := c.Delete(context.Background(), obj); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
