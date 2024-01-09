package main

import (
	"attacknet/cmd/pkg"
	"attacknet/cmd/pkg/plan"
	"attacknet/cmd/pkg/project"
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
	Plan struct {
		Name string `arg:"" optional:"" name:"name" help:"The name of the test suite to be generated"`
		Path string `arg:"" optional:"" type:"existingfile" name:"path" help:"Location of the planner configuration."`
	} `cmd:"" help:"Construct an attacknet suite for a client"`
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
		err = project.InitializeProject(dir, CLI.Init.Force)
		if err != nil {
			log.Fatal(err)
		}
	case "init <path>":
		err := project.InitializeProject(CLI.Init.Path, CLI.Init.Force)
		if err != nil {
			log.Fatal(err)
		}
	case "start <suite name>":
		ctx, cancelCtxFunc := context.WithCancel(context.Background())
		defer cancelCtxFunc()
		cfg, err := project.LoadSuiteConfigFromName(CLI.Start.Suite)
		if err != nil {
			log.Fatal(err)
		}
		err = pkg.StartTestSuite(ctx, cfg)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	case "plan <path>":
		config, err := plan.LoadPlannerConfigFromPath(CLI.Plan.Path)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		err = plan.BuildPlan(CLI.Plan.Name, config)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}
}
