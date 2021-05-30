package podTimeController

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Controller holds properties and components of the controller.
type Controller struct {
	log            log.StdLogger // From logrus, but can be replaced with log from the standard library
	kubeConfigPath string
	kubeClient     kubernetes.Interface
	// queue        workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer // stores a PodInformer
}

// Run initializes and executes the controller loop.
func (c *Controller) Run(kubeConfigPath string) error {
	err := c.createKubeClient(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("Error creating KubeConfig: %v\n", err)
	}

	// Create a shared informer factory that refreshes its cache every minute.
	factory := informers.NewSharedInformerFactory(c.kubeClient, 60*time.Second)
	// Use the factory to create a Pod informer, to watch pods.
	c.informer = factory.Core().V1().Pods().Informer()

	// Create a channel that will exit when a signal is received or if the
	// informer fails to initialize its cache.
	// quit := c.createSignalHandler()
	stop := make(chan struct{})
	// Create another channel that receives SIGTERM and SIGINT signals and triggers cleanup and exit.
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-ch
		// I can not access the log memver of the Controller type from this routine so
		// I use the default imported log.
		log.Printf("received signal %s, exiting...\n", sig)
		close(stop)
	}()

	factory.Start(stop)

	// Wait for the informer cache to sync.
	cacheSynced := cache.WaitForCacheSync(stop, c.informer.HasSynced)
	if !cacheSynced {
		return fmt.Errorf("error while waiting for informer cache to sync")
	}

	// Register individual functions with the informer event handler.
	c.informer.AddEventHandler(
		&cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAddPod,
			DeleteFunc: c.handleDeletePod,
			UpdateFunc: c.handleUpdatePod,
		})

	// Wait for the quit channel to be done.
	<-stop
	return nil
}
