// Command-line interface for the pod time controller.
package main

import (
	"flag"
	"fmt"
	"os"
	"podTimeController"

	log "github.com/sirupsen/logrus"
)

func main() {
	fs := flag.NewFlagSet("pod-time-controller", flag.ExitOnError)
	fs.SetOutput(os.Stderr)

	debug := fs.Bool("debug", false, "Enable debug logging")
	showVersion := fs.Bool("version", false, "Display the version and git commit")
	kubeConfigPath := fs.String("kubeconfig", "", "Path to a KubeConfig file (defaults to none which will use an in-cluster KubeConfig)")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	if *showVersion {
		fmt.Fprintf(os.Stdout, "pod-time-controller version %s (git commit %s)\n", podTimeController.Version, podTimeController.GitCommit)
		os.Exit(0)
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	var controller podTimeController.Controller
	err = controller.Run(*kubeConfigPath)
	if err != nil {
		log.Fatalf("%v", err.Error())
	}
}
