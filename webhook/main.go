package main

import (
	"os"

	"github.com/DelfiaProducts/docp-agent-k8s/pkg"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sWebhook := pkg.NewK8sWebhook(port, logger)
	if err := k8sWebhook.Start(); err != nil {
		logger.Error("docp webhook", "error", err.Error())
	}
}
