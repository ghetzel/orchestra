//go:build wasm
// +build wasm

package main

//go:generate vugugen -s -r

import (
	"fmt"

	"flag"

	"github.com/vugu/vugu"
	"github.com/vugu/vugu/domrender"
)

func main() {

	mountPoint := flag.String("mount-point", "#vugu_mount_point", "The query selector for the mount point for the root component, if it is not a full HTML component")
	flag.Parse()

	fmt.Printf("Entering main(), -mount-point=%q\n", *mountPoint)
	defer fmt.Printf("Exiting main()\n")

	renderer, err := domrender.New(*mountPoint)
	if err != nil {
		panic(err)
	}
	defer renderer.Release()

	buildEnv, err := vugu.NewBuildEnv(renderer.EventEnv())
	if err != nil {
		panic(err)
	}

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		var message = typeutil.String(r)
	// 		var stacktrace = string(debug.Stack())

	// 		global.App.ShowCrashAlert(message, stacktrace)
	// 	}
	// }()
	var rootBuilder = new(Root)

	for ok := true; ok; ok = renderer.EventWait() {
		var buildResults = buildEnv.RunBuild(rootBuilder)

		if err := renderer.Render(buildResults); err != nil {
			panic(err)
		}
	}
}
