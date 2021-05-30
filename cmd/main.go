package main

import (
	"podTimeController"

	log "github.com/sirupsen/logrus"
)

func main() {
	var controller podTimeController.Controller
	err := controller.Run("./kubeconfig")
	if err != nil {
		log.Fatalf("%v", err.Error())
	}
}
