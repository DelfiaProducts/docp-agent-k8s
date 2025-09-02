package main

import (
	"os"

	"github.com/DelfiaProducts/docp-agent-k8s/pkg"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
)

func main() {
	logger := utils.NewK8sLoggerText(os.Stdout)
	k8sUpdater := pkg.NewK8sUpdater(logger)
	if err := k8sUpdater.Start(); err != nil {
		logger.Error("docp updater", "error", err.Error())
	}
}
