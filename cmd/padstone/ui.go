package main

import (
	"github.com/mitchellh/cli"
	tfcmd "github.com/hashicorp/terraform/command"
)

type UI struct {
	*cli.ConcurrentUi
	*tfcmd.UIInput
}
