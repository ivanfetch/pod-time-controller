package podTimeController

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"

	v1 "k8s.io/api/core/v1"
)

func (c *Controller) handleAddPod(object interface{}) {
	// Type-assert the interface to a Pod, to avoid potentially panicing.
	pod, valid := object.(*v1.Pod)
	if !valid {
		log.Errorf("Error converting object to a Pod: %+V\n", object)
		return
	}
	log.Debugf("new pod %s\n", pod.Name)

	if hasAnnotation(pod, triggerAnnotationName) && !hasAnnotation(pod, timeAnnotationName) {
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(object)
		if err != nil {
			log.Errorf("Error making key for pod object %v: %v\n", pod.Name, err)
		} else {
			log.Infof("pod added: adding key to queue: %v", key)
			c.queue.Add(key)
		}
	}
	return
}

func (c *Controller) handleDeletePod(object interface{}) {
	// Type-assert the interface to a Pod, to avoid potentially panicing.
	pod, valid := object.(*v1.Pod)
	if !valid {
		log.Errorf("Error converting object to a Pod: %+V\n", object)
		return
	}
	log.Debugf("deleted pod %s\n", pod.Name)
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(object)
	if err != nil {
		log.Errorf("Error making key for pod object %v: %v\n", pod.Name, err)
	} else {
		log.Infof("attempting to remove key from queue: %v", key)
		c.queue.Forget(key)
		c.queue.Done(key)
	}
	return
}

func (c *Controller) handleUpdatePod(oldObject, newObject interface{}) {
	// Type-assert the interface to a Pod, to avoid potentially panicing.
	// It doesn't matter what the pre-updated object was, in this case, so use
	// the new object.
	pod, valid := newObject.(*v1.Pod)
	if !valid {
		log.Errorf("Error converting object to a Pod: %+V\n", newObject)
		return
	}
	log.Debugf("updated pod %s\n", pod.Name)

	if hasAnnotation(pod, triggerAnnotationName) && !hasAnnotation(pod, timeAnnotationName) {
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(newObject)
		if err != nil {
			log.Errorf("Error making key for pod object %v: %v\n", pod.Name, err)
		} else {
			log.Infof("pod updated: adding key to queue: %v", key)
			c.queue.Add(key)
		}
	}
	return
}

// hasAnnotation accepts a pod object, and returns true of it has the
// timeAnnotation annotation.
func hasAnnotation(pod *v1.Pod, annotationName string) bool {
	annotations := pod.ObjectMeta.Annotations
	_, annotationExists := annotations[annotationName]
	if !annotationExists {
		log.Debugf("pod %s does not have annotation %s", pod.ObjectMeta.Name, annotationName)
		return false
	}
	log.Debugf("pod %s has annotation %s", pod.ObjectMeta.Name, annotationName)
	return true
}
