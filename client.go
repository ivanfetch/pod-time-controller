package podTimeController

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClient uses the kubeConfigPath member to create a Kubernetes client
// interface. If the KubeConfig is blank, an in-cluster KubeConfig
// will be used. This allows running in or out of cluster.
func (c *Controller) createKubeClient(kubeConfigPath string) error {
	var kubeConfig *rest.Config
	var err error

	if kubeConfigPath != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return fmt.Errorf("unable to load kubeconfig from %s: %v", kubeConfigPath, err)
		}
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("unable to load in-cluster config: %v", err)
		}
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("unable to create a client: %v", err)
	}

	c.kubeClient = client
	return nil
}
