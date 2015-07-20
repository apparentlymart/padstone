package main

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

type UIHook struct {
	terraform.NilHook

	ui *UI
	verbose bool
}

func (h *UIHook) PreApply(instance *terraform.InstanceInfo, istate *terraform.InstanceState, diff *terraform.InstanceDiff) (terraform.HookAction, error) {
	if diff.Destroy || diff.DestroyTainted {
		h.ui.Info(fmt.Sprintf("[%v] Destroying ...", instance.HumanId()))
	} else {
		h.ui.Info(fmt.Sprintf("[%v] Creating...", instance.HumanId()))
	}
	return terraform.HookActionContinue, nil
}

func (h *UIHook) PostApply(instance *terraform.InstanceInfo, istate *terraform.InstanceState, err error) (terraform.HookAction, error) {
	if err != nil {
		h.ui.Error(fmt.Sprintf("[%v] Error during apply: %v", instance.HumanId(), err.Error()))
	} else {
		if h.verbose {
			if istate.ID != "" {
				h.ui.Info(fmt.Sprintf("[%v] Successfully created as %v", instance.HumanId(), istate.ID))
			} else {
				h.ui.Info(fmt.Sprintf("[%v] Successfully destroyed", instance.HumanId()))
			}
		}
	}
	return terraform.HookActionContinue, nil
}

func (h *UIHook) PreProvisionResource(instance *terraform.InstanceInfo, istate *terraform.InstanceState) (terraform.HookAction, error) {
	h.ui.Info(fmt.Sprintf("[%v] Provisioning...", instance.HumanId()))
	return terraform.HookActionContinue, nil
}

func (h *UIHook) PostProvisionResource(instance *terraform.InstanceInfo, istate *terraform.InstanceState) (terraform.HookAction, error) {
	if h.verbose {
		h.ui.Info(fmt.Sprintf("[%v] Successfully provisioned", instance.HumanId()))
	}
	return terraform.HookActionContinue, nil
}

func (h *UIHook) ProvisionOutput(instance *terraform.InstanceInfo, name string, line string) {
	h.ui.Info(fmt.Sprintf("[%v %v] %v", instance.HumanId(), name, line))
}
