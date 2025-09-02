package main

import (
	"os"

	"github.com/DelfiaProducts/docp-agent-k8s/pkg"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
)

func main() {
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sManager := pkg.NewK8sManager(logger)
	if err := k8sManager.Start(); err != nil {
		logger.Error("docp manager", "error", err.Error())
	}
}
