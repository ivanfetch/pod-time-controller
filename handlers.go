package podTimeController

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
)

func (c Controller) handleAddPod(object interface{}) {
	// Type-assert the interface to a Pod, to avoid potentially panicing?
	pod, valid := object.(*v1.Pod)
	if !valid {
		log.Errorf("Error converting object to a Pod: %+V\n", object)
		return
	}
	fmt.Printf("Added pod %s\n", pod.Name)
	return
}

func (c Controller) handleDeletePod(object interface{}) {
	// Type-assert the interface to a Pod, to avoid potentially panicing?
	pod, valid := object.(*v1.Pod)
	if !valid {
		log.Errorf("Error converting object to a Pod: %+V\n", object)
		return
	}
	fmt.Printf("Deleted pod %s\n", pod.Name)
	return
}

func (c Controller) handleUpdatePod(oldObject, newObject interface{}) {
	// Type-assert the interface to a Pod, to avoid potentially panicing?
	// It doesn't matter what the pre-updated object was, in this case.
	pod, valid := newObject.(*v1.Pod)
	if !valid {
		log.Errorf("Error converting object to a Pod: %+V\n", newObject)
		return
	}
	fmt.Printf("Updated pod %s\n", pod.Name)
	return
}
