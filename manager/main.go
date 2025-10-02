package main

import (
	"os"

	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

func main() {
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sManager := pkg.NewK8sManager(logger)
	if err := k8sManager.Start(); err != nil {
		logger.Error("orya manager", "error", err.Error())
	}
}
