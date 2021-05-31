package podTimeController

import (
	"fmt"

	log "github.com/sirupsen/logrus"

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
		log.Debugf("creating KubeConfig using file %s", kubeConfigPath)
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return fmt.Errorf("unable to load kubeconfig from %s: %v", kubeConfigPath, err)
		}
	} else {
		log.Debugf("creating in-cluster KubeConfig, as no KubeConfigPath was specified")
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("unable to load in-cluster KubeConfig, if testing this controller you can use the -kubeconfig option to specify a KubeConfig file. Error: %v", err)
		}
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("unable to create a client: %v", err)
	}

	c.kubeClient = client
	return nil
}
