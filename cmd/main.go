package main

import (
	"podTimeController"

	log "github.com/sirupsen/logrus"
)

func main() {
	err := podTimeController.Run()
	if err != nil {
		log.Fatalf("%v", err.Error())
	}
}
