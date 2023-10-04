package main

import (
	"attacknet/cmd/pkg"
	"context"
	"github.com/alecthomas/kong"
	"log"
	"os"
)

var CLI struct {
	Init struct {
		Force bool   `arg:"force" optional:"" default:"false" name:"force" help:"Overwrite existing project"`
		Path  string `arg:"" optional:"" type:"existingdir" name:"path" help:"Path to initialize project on. Defaults to current working directory."`
	} `cmd:"" help:"Initialize an attacknet project"`
	Start struct {
		Suite string `arg:"" name:"suite name" help:"The test suite to run. These are located in ./test-suites"`
	} `cmd:"" help:"Run a specified test suite"`
}

func main() {
	// todo: use flag for arg parse

	c := kong.Parse(&CLI)

	b := c.Command()
	switch b {
	case "init":
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		err = pkg.InitializeProject(dir, CLI.Init.Force)
		if err != nil {
			log.Fatal(err)
		}
	case "init <path>":
		err := pkg.InitializeProject(CLI.Init.Path, CLI.Init.Force)
		if err != nil {
			log.Fatal(err)
		}
	case "start <suite name>":
		ctx, cancelCtxFunc := context.WithCancel(context.Background())
		defer cancelCtxFunc()
		err := pkg.StartTestSuite(ctx, CLI.Start.Suite)
		if err != nil {
			log.Fatal(err)
		}
	}
}
