package main

import (
	"clusterH/clusterDO"
	"clusterH/clusterInterface"
	"clusterH/clusterLocal"
	"clusterH/store"

	"github.com/urfave/cli"
)

func main() {
	defer store.Close()

	currentClusterType := store.GetCurrentClusterType()
	var currentInterface []cli.Command

	if currentClusterType == "local" {
		currentInterface = clusterLocal.GetUI()
	} else if currentClusterType == "do" {
		currentInterface = clusterDO.GetUI()
	}

	clusterInterface.Run(currentInterface)
}
