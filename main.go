package main

import (
	"github.com/taskcluster/tc-tui/controller"
)

func main() {
	ctrl := controller.NewController()

	if err := ctrl.StartUI(); err != nil {
		panic(err)
	}
}
