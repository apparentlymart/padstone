package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/cli"

	tfcmd "github.com/hashicorp/terraform/command"

	"github.com/mitchellh/colorstring"
)

func main() {
	ui := &UI{
		ConcurrentUi: &cli.ConcurrentUi{
			Ui: &cli.BasicUi{
				Reader:      os.Stdin,
				Writer:      os.Stdout,
				ErrorWriter: os.Stderr,
			},
		},
		UIInput: &tfcmd.UIInput{
			Colorize: &colorstring.Colorize{
				Colors:  colorstring.DefaultColors,
				Disable: true,
			},
		},
	}

	logFile, err := os.OpenFile("padstone.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		ui.Error(fmt.Sprintf("Error opening log file: %s", err))
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	clParser := flags.NewParser(nil, flags.Default)
	clParser.AddCommand(
		"build",
		"Run a build",
		"The 'build' command creates new resources from a configuration",
		&BuildCommand{
			ui: ui,
		},
	)
	clParser.AddCommand(
		"destroy",
		"Destroy the results of a build",
		"The 'destroy' command destroys the resources from an earlier build",
		&DestroyCommand{
			ui: ui,
		},
	)

	if _, err := clParser.Parse(); err != nil {
		os.Exit(1)
	}
}
