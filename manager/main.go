package main

import (
	"os"

	"github.com/OryaHub/agent-k8s/agents"
	"github.com/OryaHub/agent-k8s/utils"
)

func main() {
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sManager := agents.NewK8sManager(logger)
	if err := k8sManager.Start(); err != nil {
		logger.Error("orya manager", "error", err.Error())
	}
}
