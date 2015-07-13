package main

import (
	"fmt"

	tfcmd "github.com/hashicorp/terraform/command"
	tfmodcfg "github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"

	"github.com/mitchellh/colorstring"

	"github.com/apparentlymart/padstone/padstone"
)

func main() {
	sysConfig := BuiltinConfig
	sysConfig.Discover()

	config, err := padstone.LoadConfig(".")
	if err != nil {
		fmt.Println(err)
		return
	}

	storage := &tfmodcfg.FolderStorage{
		StorageDir: ".padstone",
	}

	uiInput := &tfcmd.UIInput{
		Colorize: &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
		},
	}

	state := terraform.NewState()

	uiHook := &UIHook{}
	stateHook := &StateHook{}

	ctx := &padstone.Context{
		Config:        config,
		State:         state,
		Providers:     sysConfig.ProviderFactories(),
		Provisioners:  sysConfig.ProvisionerFactories(),
		Variables:     map[string]string{},
		Hooks:         []terraform.Hook{stateHook, uiHook},
		UIInput:       uiInput,
		ModuleStorage: storage,
	}

	warns, errs := ctx.Validate()
	for _, warning := range warns {
		fmt.Printf("Warning: %v\n", warning)
	}
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}

	err = ctx.Build()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("\n \u2713 Build succeeded! Now destroying temporary resources...\n")

	err = ctx.CleanUp()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = WriteState(ctx.ResultState)
	if err != nil {
		fmt.Println(err)
		return
	}

	outputs := ctx.ResultState.Modules[0].Outputs
	if len(outputs) > 0 {
		fmt.Println("\nOutputs:")
		for k, v := range outputs {
			fmt.Printf("- %v = %v\n", k, v)
		}
		fmt.Println("")
	}
}
