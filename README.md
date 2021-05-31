# Pod Timestamp Controller

This Kubernetes controller

* Listens for new or updated pods
* Acts on pods with an `addtime` annotation set to any value
* Adds a `timestamp` annotation with the current date and time

This is my foray into using Go for Kubernetes programming, and this project currently represents a balance of quality and proof-of-concept / "get something out the door." :)

So far I find the Kubernetes related Go packages are slower to learn, as there are abstractions upon abstractions, and many Go Interfaces passed between functions. For more details, see the "Approach" section below.

## Example Output

Logs from within Kubernetes will have full log timestamps instead of the below time offsets since the program was started.

```
INFO[0000] pod time controller starting - pods with the addtime annotation, will be annotated with timestamp set to the current date and time 
```

A Kubernetes Deployment contains the `addtime` annotation in the pod-spec. After I updated this deployment, forcing it to create two new pods, the controller detects those pods.

```
INFO[0015] pod added: adding key to queue: default/yourapp-584d765d48-hgvcb 
INFO[0015] Annotating namespace "default", pod "yourapp-584d765d48-hgvcb" with timestamp 2021-05-30T21:30:23-04:00 
INFO[0018] pod added: adding key to queue: default/yourapp-584d765d48-t66z5 
INFO[0018] Annotating namespace "default", pod "yourapp-584d765d48-t66z5" with timestamp 2021-05-30T21:30:26-04:00 
```

The controller then receives update events for the same new pods, which are added to the queue again because the pods do not yet show the `timestamp` annotation (previously already added by the controller). I believe this is a race condition related to the Kubernetes Informer cache?

```
INFO[0018] pod updated: adding key to queue: default/yourapp-584d765d48-t66z5 
INFO[0018] Annotating namespace "default", pod "yourapp-584d765d48-t66z5" with timestamp 2021-05-30T21:30:26-04:00
```

Next the controller detects the deleted pods which were recycled by the Deployment, and attempts to clean them up  from the controllers worker queue (note in this case, these pods were never in the queue).

```
INFO[0031] attempting to remove key from queue: default/yourapp-688477cb9f-l2xcd 
INFO[0031] attempting to remove key from queue: default/yourapp-688477cb9f-sdbh2 
```

## Usage

The included Helm chart can be used to install this controller and pull its image from the Docker Hub.

```bash
helm upgrade --install --create-namespace -n pod-time-controller pod-time-controller charts/pod-time-controller
```

Create a test pod, then add the `addtime` annotation so this controller will act on that pod and add the `timestamp` annotation.

```bash
kubectl run -n default --image nginx testpod
```

```bash
kubectl annotate -n default pod/testpod addtime=true
```

Then use `kubectl logs --namespace pod-time-controller -l "app.kubernetes.io/name=pod-time-controller,app.kubernetes.io/instance=pod-time-controller"` to see the controller logs, which should show the `testpod` being annotated with the time stamp.

Clean up the testpod and controller:

```bash
kubectl delete pod -n default testpod
helm delete -n pod-time-controller pod-time-controller
kubectl delete namespace pod-time-controller
```

## Building and Testing

This repository includes a [Makefile](./Makefile) to ease common tasks.

* `make test` - Format, vet, and test code
* `make build` - Build locally including setting of the version (Git tag) and Git commit variables used by the `pod-time-controller -version` option
* `make docker-build` - Build a local Docker image
* `make docker-push` - Push a local Docker image to a registry (requires editing variables at the top of the Makefile)

## Future Updates and Considerations

* Add liveness/readiness probes to the controller pod? These are best-practice for Kubernetes-hosted applications, but an answered HTTP request doesn't prove that controller pods are functional - I'm not sure what validating work a probe should do without being too expensive (like talking to the Kube API for every probe).
* Add leader election so multiple pods can be run without causing undo work for the Kube API.
* Switch [main.go](./cmd/main.go) to have less code and to instead call a CLI function, perhaps using [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) to process command-line options plus environment variables.
* `Controller.Run()` also initializes the global logger, Kubernetes client, Kubernetes Informer, and work queue. These things might be better handled in a constructor, with [functional options](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis) to pass the log-level, KubeConfig, Etc.
* Make the Informer cache refresh time configurable - 1 minute was used as a proof-of-concept.
* Perhaps decompose some functions into smaller ones for readability, such as `Controller.Run()`.
* Is there a better way to turn a key pulled from the worker queue into its namespace and pod-name to patch that pod in Kube?
	* I currently split the key on slash (/) to separate the namespace, to be able to call `kubeClient.CoreV1.Pods(NamespaceName).Patch(...)` with a separate namespace and pod name.
	* I recall seeing a note in the worker queue source code that there is a goal to store full objects in the queue instead of requiring conversion from/to these string-based keys...
* Perhaps the logger should be a controller struct member?

## Approach

Here are some notes about how this controller works, including some of my thought process as I looked at other controller code on the Internet along with the Kubernetes Go packages.

Here is a high level view of what this code does:

* Use a "shared informer"to watch and cache events from the Kubernetes API about Pod resources being added, updated, or deleted.
* Call Go functions when those events are received, which determine whether the controller should act on that pod - if so, the pod is added to a worker queue. Delete events remove the pod from the queue.
* In parallel, a Go routine processes items from the worker queue, adding a `timestamp` annotation to those pods (because that's what this controller does). 

### What About Frameworks or Generators?

There are things like [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) that help with controllers and your own custom Kube resources, but

* I want to be capable using the Kubernetes Go packages
* For this project, Kubebuilder seems like a hammer for pushing in a thumbtack

### What Makes a Controller?

A controller executes its loop-of-code, watching Kubernetes for events its interested in. In this case, when this controller sees a Pod with an annotation `addtime`, it will add an annotation `timestamp`.

* I need an "informer" in Kubernetes Go - this helps me watch a particular Kube resource more efficiently by including a cache which stores a list of resources in memory, instead of only using a Kube Watch resource and hoping that no events are missed from the Kube API. The cache is local, and optionally refreshable.
* There is a [tools/cache package](https://pkg.go.dev/k8s.io/client-go/tools/cache) which is described by the Kubernetes client-go package docs as "useful for writing controllers." There are informer types and many other useful things in this package, and a lot of the Internet seems to use underlying functions from here as the basis for controllers, but see the next bullet.
* A [shared informer factory](https://github.com/kubernetes/client-go/blob/v0.21.1/informers/factory.go#L96) helps informers to be shared - meaning that the same cache (index) can be used with multiple sets of handler functions (the Go functions that act on events from Kubernetes). I don't quite need my informer to be shared, but it appears to be the standard and the way forward for a more efficient controller (from the perspective of the kube API and code). Also, who wants an informer when you could have ... an informer factory!? :)

### Watching For Pods

* To tell a shared informer what to inform about, I use a [PodInformer type](https://github.com/kubernetes/client-go/blob/v0.21.1/informers/core/v1/pod.go#L37) which watches for events on Pods in all namespaces.
* Having looked at enough mostly-trusted Kubernetes controller code on the Internet, I knew to wait for the informer cache to sync, before my controller should do anything else. To do that I use [WaitForCacheSync](https://github.com/kubernetes/client-go/blob/v0.21.1/tools/cache/shared_informer.go#L254), which wants a Go channel, why?

### When to Stop Looping?

* The informer needs to know when to stop watching, informing, and caching, for which it uses a Go channel.
* The same channel can also be triggered when an operating system signal is received, like `SIGKILL` when this Kubernetes controller pod is terminated.
* Over all, this channel represents when things should stop processing and clean up; prepare to exit.
* I borrowed some code from [the Kubernetes example controller signal handler function](https://github.com/kubernetes/sample-controller/blob/master/pkg/signals/signal.go#L29) - I 90% understand what this code is doing, I have not done much with channels in Go.

### Running Code On Kubernetes Pod Events

* I want to register the Go functions to be called when Pods are added, updated, or deleted. For this I use `AddEventHandler` via the informer object.
* This registration involves a [ResourceEventHandler type](https://pkg.go.dev/k8s.io/client-go@v0.21.1/tools/cache#ResourceEventHandler) which states in its documentation:
	* "The handlers MUST NOT modify the objects received; this concerns not only the top level of structure but all the data structures reachable from it."
* Therefore, the handler functions will **only** determine whether a Pod has the "trigger annotation" which indicates that Pod would like this controller to add a `timestamp` annotation to it. Where does the work of adding that annotation happen?

### Actually Annotating Pods

* Pods that should have a new annotation, will be added to a worker queue, provided by [this Kubernetes client-go rate-limiting work queue package](https://pkg.go.dev/k8s.io/client-go/util/workqueue#RateLimitingInterface).
	* This queue will use the same Go channel to determine when to stop the Go routines that process the queue.
* What gets added to the queue exactly? The Pod key, consisting of `namespace/name`. This is obtained using the [cache.DeletionHandlingMetaNamespaceKeyFunc](https://github.com/kubernetes/client-go/blob/v0.21.1/tools/cache/controller.go#L294) function which takes the Pod deletion state into account before returning the key. Can I just say ... `DeletionHandlingMetaNamespaceKeyFunc` is quite a name for a function!
* The queue has a [Get method](https://github.com/kubernetes/client-go/blob/v0.21.1/util/workqueue/queue.go#L147) that blocks until there is an item on the queue. This function returns a second boolean argument to indicate when the queue wants to shutdown, determined by the Go channel.

### Conclusion

The above is a lot of explanation, linking to Go documentation for packages that can feel very complex. In the interest of time, I haven't continued to refactor my code, but I hope it is understandable with the aide of this background and my understanding of the Go packages for controllers.
