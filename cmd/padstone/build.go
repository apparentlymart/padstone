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

type BuildCommand struct {
	ui    *UI
	input *tfcmd.UIInput

	Verbose bool             `short:"v" long:"verbose" description:"show detailed information about resources"`
	Args    BuildCommandArgs `positional-args:"true" required:"true"`
}

type BuildCommandArgs struct {
	ConfigDir string   `positional-arg-name:"config-dir" description:"path to the directory containing the build configuration"`
	StateFile string   `positional-arg-name:"state-output-file" description:"path where the resulting state file will be written"`
	VarSpecs  []string `positional-args:"true" positional-arg-name:"varname=value" description:"zero or more explicit variable value specifications"`
}

func (c *BuildCommand) Execute(args []string) error {
	sysConfig := BuiltinConfig
	sysConfig.Discover()

	config, err := padstone.LoadConfig(c.Args.ConfigDir)
	if err != nil {
		return err
	}

	storage := &tfmodcfg.FolderStorage{
		StorageDir: ".padstone",
	}

	// The state file must not already exist, since we don't to
	// accidentally clobber the record of resources created in an
	// earlier build.
	_, err = os.Lstat(c.Args.StateFile)
	if err == nil {
		return fmt.Errorf("state file %s already exists; specify a different name or destroy it with 'padstone destroy' before generating a new set of resources", c.Args.StateFile)
	}

	state := terraform.NewState()

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

	err = ctx.Build()
	if err != nil {
		return err
	}

	c.ui.Info("--- Build succeeded! Now destroying temporary resources... ---")

	err = ctx.CleanUp()
	if err != nil {
		return err
	}

	_, err = stateHook.PostStateUpdate(ctx.ResultState)
	if err != nil {
		return err
	}

	outputs := ctx.ResultState.Modules[0].Outputs
	if len(outputs) > 0 {
		c.ui.Output("\nOutputs:")
		for k, v := range outputs {
			c.ui.Output(fmt.Sprintf("- %v = %v", k, v))
		}
		c.ui.Output("")
	}

	return nil
}
