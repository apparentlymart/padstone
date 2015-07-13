package main

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

type UIHook struct {
	terraform.NilHook
}

func (h *UIHook) PreApply(instance *terraform.InstanceInfo, istate *terraform.InstanceState, diff *terraform.InstanceDiff) (terraform.HookAction, error) {
	if diff.Destroy || diff.DestroyTainted {
		fmt.Printf("Destroying %v...\n", instance.HumanId())
	} else {
		fmt.Printf("Creating %v...\n", instance.HumanId())
	}
	return terraform.HookActionContinue, nil
}

func (h *UIHook) PostApply(instance *terraform.InstanceInfo, istate *terraform.InstanceState, err error) (terraform.HookAction, error) {
	if err != nil {
		fmt.Printf("Error for %v: %v\n", instance.HumanId(), err.Error())
	} else {
		if istate.ID != "" {
			fmt.Printf("Successfully created %v as %v\n", instance.HumanId(), istate.ID)
		} else {
			fmt.Printf("Successfully destroyed %v\n", instance.HumanId())
		}
	}
	return terraform.HookActionContinue, nil
}

func (h *UIHook) PreProvisionResource(instance *terraform.InstanceInfo, istate *terraform.InstanceState) (terraform.HookAction, error) {
	fmt.Printf("Provisioning %v...\n", instance.HumanId())
	return terraform.HookActionContinue, nil
}

func (h *UIHook) PostProvisionResource(instance *terraform.InstanceInfo, istate *terraform.InstanceState) (terraform.HookAction, error) {
	fmt.Printf("Successfully provisioned %v\n", instance.HumanId())
	return terraform.HookActionContinue, nil
}

func (h *UIHook) ProvisionOutput(instance *terraform.InstanceInfo, name string, line string) {
	fmt.Printf("[%v %v] %v\n", instance.HumanId(), name, line)
}
