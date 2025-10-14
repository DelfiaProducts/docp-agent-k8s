package main

import (
	"os"

	"github.com/OryaHub/agent-k8s/agents"
	"github.com/OryaHub/agent-k8s/utils"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sWebhook := agents.NewK8sWebhook(port, logger)
	if err := k8sWebhook.Start(); err != nil {
		logger.Error("orya webhook", "error", err.Error())
	}
}
