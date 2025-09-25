package main

import (
	"os"

	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "12012"
	}
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sAgent := pkg.NewK8sAgent(port, logger)
	if err := k8sAgent.Start(); err != nil {
		logger.Error("orya agent", "error", err.Error())
	}
}
