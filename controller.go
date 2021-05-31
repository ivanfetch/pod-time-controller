package podTimeController

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"os"
	"os/signal"
	"syscall"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds properties and components of the controller.
type Controller struct {
	kubeConfigPath string
	kubeClient     kubernetes.Interface
	queue          workqueue.RateLimitingInterface
	informer       cache.SharedIndexInformer // stores a PodInformer
}

// These are populated by the build process.
var Version string = "unknown"
var GitCommit string = "unknown"

const (
	triggerAnnotationName = "addtime"   // Annotation required to exist for the controller to annotate pods.
	timeAnnotationName    = "timestamp" // Annotation to add to pods
	maxQueueRetries       = 5           // max requeues for an item that fails to be patched
)

// Run initializes and executes the controller loop.
func (c *Controller) Run(kubeConfigPath string) error {
	log.SetOutput(os.Stdout)

	log.Infof("pod time controller starting - pods with the %s annotation, will be annotated with %s set to the current date and time", triggerAnnotationName, timeAnnotationName)

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
	// The owrk queue also uses this channel.
	stop := createSignalHandler()

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

	c.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Run a worker to process items on the work queue every second.
	go wait.Until(c.runWorker, time.Second, stop)

	// Wait for the stop channel to be done.
	<-stop
	return nil
}

// runWorker loops to process items on the queue until the queue determins it
// must be shutdown, via the channel.
func (c *Controller) runWorker() {
	for c.processNextQueueItem() {
		// processNextQueueItem() does the work, here we keep looping. . .
	}
}

// processNextQueueItemprocesses the next item from the queue, and re-queues
// it if processing is not possible. This returns false when the queue should
// shutdown.
func (c *Controller) processNextQueueItem() bool {
	// Get() will block until there is an item to be processed.
	// Get() returns done==true if this routine should exit, determined via the
	// channel that the queue is attached to.
	key, shutdown := c.queue.Get()

	if shutdown {
		return false
	}
	// the queue returns an interface but key is a string.
	keyString := fmt.Sprintf("%v", key)
	log.Debugf("processing key %s from the queue", keyString)

	// There may be a better way to turn this key into something addressable in
	// the Kube API, but this does work and is pretty strait-forward.
	var namespace, podName string
	if strings.Contains(keyString, "/") {
		keySplit := strings.Split(keyString, "/")
		namespace = keySplit[0]
		podName = keySplit[1]
	} else {
		podName = keyString
	}

	err := c.annotatePod(namespace, podName)
	if err == nil {
		c.queue.Forget(key) // for the queue rate limiter
		c.queue.Done(key)
		return true
	} else {
		// THere was an error
		if c.queue.NumRequeues(key) < maxQueueRetries {
			log.Errorf("requeueing after error patching %s : %v", key, err)
			c.queue.AddRateLimited(key)
		} else {
			log.Errorf("error patching %s after max %d number of retries: %v", key, maxQueueRetries, err)
			c.queue.Forget(key) // for the queue rate limiter
			c.queue.Done(key)
		}
	}

	return true
}

// annotatePod accepts a namespace name and pod name, adding the
// `timeAnnotationName` annotation to that pod.
func (c Controller) annotatePod(podNamespace, podName string) error {
	log.Infof("Annotating namespace %q, pod %q", podNamespace, podName)
	podPatch := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`,
		timeAnnotationName,
		time.Now().Format(time.RFC3339))
	log.Debugf("patch for pod %s/%s is: %v", podNamespace, podName, podPatch)

	_, err := c.kubeClient.CoreV1().Pods(podNamespace).Patch(context.TODO(), podName, types.StrategicMergePatchType, []byte(podPatch), metav1.PatchOptions{})

	if err != nil {
		return fmt.Errorf("error while patching namespace %q pod %q: %v", podNamespace, podName, err)
	}
	return nil
}

// createSignalHandler creates a channel that the Kubernetes Informer and
// controller worker queue will use to determine when they need to shutdown.
// This channel will also be impacted if a signal is received (SIGTERM).
func createSignalHandler() (stopChannel <-chan struct{}) {
	stop := make(chan struct{})
	// Create another channel that receives SIGTERM and SIGINT signals and triggers cleanup and exit.
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-ch
		log.Infof("received signal %s, exiting...\n", sig)
		close(stop)
	}()
	return stop
}
