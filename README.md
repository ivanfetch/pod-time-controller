# Pod Timestamp Controller

This Kubernetes controller

* Listens for new or updated pods
* Acts on pods with an `addtime` annotation set to any value
* Adds a `timestamp` annotation with the current date and time

This is my foray into using Go for Kubernetes programming, and this project currently represents a balance of quality and proof-of-concept / "get something out the door." :)

So far I find the Kubernetes related GO packages are slower to learn, as there are abstractions upon abstractions, and many Go Interfaces passed between functions.

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
