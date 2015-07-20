package main

import (
	"github.com/hashicorp/terraform/terraform"
)

type StateHook struct {
	terraform.NilHook

	OutputFilename string
}

func (h *StateHook) PostStateUpdate(state *terraform.State) (terraform.HookAction, error) {
	err := WriteState(state, h.OutputFilename)
	if err != nil {
		return terraform.HookActionHalt, err
	}
	return terraform.HookActionContinue, nil
}
