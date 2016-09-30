package main

import (
	"clusterH/clusterDO"
	"clusterH/clusterInterface"
	"clusterH/clusterLocal"
	"clusterH/store"
	"fmt"

	"github.com/urfave/cli"
)

func main() {
	defer store.Close()

	currentClusterType := store.GetCurrentClusterType()
	var currentInterface []cli.Command

	fmt.Println("=====>", currentClusterType)

	if currentClusterType == "local" {
		currentInterface = clusterLocal.GetUI()
	} else if currentClusterType == "do" {
		currentInterface = clusterDO.GetUI()
	}

	clusterInterface.Run(currentInterface)
}
