package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/apparentlymart/padstone/padstone"

	tfcmd "github.com/hashicorp/terraform/command"
	tfmodcfg "github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

type DestroyCommand struct {
	ui    *UI
	input *tfcmd.UIInput

	Verbose bool             `short:"v" long:"verbose" description:"show detailed information about resources"`
	Args    BuildCommandArgs `positional-args:"true" required:"true"`
}

type DestroyCommandArgs struct {
	ConfigDir string   `positional-arg-name:"config-dir" description:"path to the directory containing the build configuration"`
	StateFile string   `positional-arg-name:"state-output-file" description:"path where the resulting state file will be written"`
	VarSpecs  []string `positional-args:"true" positional-arg-name:"varname=value" description:"zero or more explicit variable value specifications"`
}

func (c *DestroyCommand) Execute(args []string) error {
	sysConfig := BuiltinConfig
	sysConfig.Discover()

	config, err := padstone.LoadConfig(c.Args.ConfigDir)
	if err != nil {
		return err
	}

	storage := &tfmodcfg.FolderStorage{
		StorageDir: ".padstone",
	}

	stateFile, err := os.Open(c.Args.StateFile)
	if err != nil {
		return fmt.Errorf("error opening state file %s: %s", c.Args.StateFile, err)
	}

	state, err := terraform.ReadState(stateFile)
	if err != nil {
		return fmt.Errorf("error reading state file %s: %s", c.Args.StateFile, err)
	}

	uiHook := &UIHook{
		ui:      c.ui,
		verbose: c.Verbose,
	}
	stateHook := &StateHook{
		OutputFilename: c.Args.StateFile,
	}

	variables := map[string]string{}
	for _, varSpec := range c.Args.VarSpecs {
		equalsIdx := strings.Index(varSpec, "=")
		if equalsIdx == -1 {
			return fmt.Errorf("variable spec %#v must be formatted as key=value", varSpec)
		}
		k := varSpec[:equalsIdx]
		v := varSpec[equalsIdx+1:]
		variables[k] = v
	}

	ctx := &padstone.Context{
		Config:        config,
		State:         state,
		Providers:     sysConfig.ProviderFactories(),
		Provisioners:  sysConfig.ProvisionerFactories(),
		Variables:     variables,
		Hooks:         []terraform.Hook{stateHook, uiHook},
		UIInput:       c.ui,
		ModuleStorage: storage,
	}

	warns, errs := ctx.Validate()
	for _, warning := range warns {
		c.ui.Warn(warning)
	}
	if len(errs) > 0 {
		for _, err := range errs {
			c.ui.Error(err.Error())
		}
		return fmt.Errorf("aborted due to configuration errors.")
	}

	err = ctx.Destroy()
	if err != nil {
		return err
	}

	rootState := ctx.State.RootModule()
	moduleCount := len(ctx.State.Modules)

	if moduleCount == 1 && len(rootState.Resources) == 0 {
		c.ui.Info("All resources destroyed")
		err := os.Remove(c.Args.StateFile)
		if err != nil {
			return fmt.Errorf("Failed to remove state file %s: %s", c.Args.StateFile, err)
		}
	} else {
		c.ui.Warn(fmt.Sprintf("Not all resources were destroyed. State file %s updated to reflect remaining resources.", c.Args.StateFile))
		_, err = stateHook.PostStateUpdate(ctx.State)
		if err != nil {
			return err
		}
	}

	return nil
}
