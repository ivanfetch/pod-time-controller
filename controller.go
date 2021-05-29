package podTimeController

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// Run initializes and executes the controller loop.
func Run() error {
	// For now hard-code to use local KubeConfig
	client, err := CreateClient("./kubeconfig")
	if err != nil {
		return fmt.Errorf("Error creating KubeConfig: %v\n", err)
	}

	// Create a shared informer factory that refreshes its cache every minute.
	factory := informers.NewSharedInformerFactory(client, 60*time.Second)
	// Use the factory to create a Pod informer, to watch pods.
	informer := factory.Core().V1().Pods().Informer()

	// Create a channel that will exit when a signal is received or if the
	// informer
	// fails to initialize its cache.
	quit := createSignalHandler()
	factory.Start(quit)

	// Wait for the informer cache to sync.
	cacheSynced := cache.WaitForCacheSync(quit, informer.HasSynced)
	if !cacheSynced {
		return fmt.Errorf("error while waiting for informer cache to sync")
	}

	// Register individual functions with the informer event handler.
	informer.AddEventHandler(
		&cache.ResourceEventHandlerFuncs{
			AddFunc:    handleAddPod,
			DeleteFunc: handleDeletePod,
			UpdateFunc: handleUpdatePod,
		})

	// Wait for the quit channel to be done.
	<-quit
	return nil
}

// createSignalHandler creates a channel that will be closed when a signal is received
func createSignalHandler() (done <-chan struct{}) {
	stop := make(chan struct{})
	// Create another channel that receives SIGTERM and SIGINT signals and triggers cleanup and exit.
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Infof("received signal %s, exiting...\n", sig)
		close(stop)
	}()
	return stop
}
