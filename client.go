package podTimeController

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClient accepts a KubeConfig filename and returns a Kubernetes client
// interface. If the KubeConfig filename is blank, an in-cluster KubeConfig
// will be used. This allows running in or out of cluster.
func CreateClient(kubeConfigPath string) (kubernetes.Interface, error) {
	var kubeConfig *rest.Config
	var err error

	if kubeConfigPath != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeConfigPath, err)
		}
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create a client: %v", err)
	}

	return client, nil
}
