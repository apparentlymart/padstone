package main

import (
	"fmt"
	"io/ioutil"

	tfcmd "github.com/hashicorp/terraform/command"
	tfremote "github.com/hashicorp/terraform/state/remote"
)

type PublishCommand struct {
	ui    *UI
	input *tfcmd.UIInput

	Args PublishCommandArgs `positional-args:"true" required:"true"`
}

type PublishCommandArgs struct {
	StateFile      string   `positional-arg-name:"state-output-file" description:"path where the resulting state file will be written"`
	StorageBackend string   `positional-arg-name:"storage-backend" description:"name of the storage backend to use for storage"`
	ConfigSpecs    []string `positional-args:"true" positional-arg-name:"key=value" description:"zero or more configuration parameters for the storage backend"`
}

func (c *PublishCommand) Execute(args []string) error {

	config, err := decodeKVSpecs(c.Args.ConfigSpecs)
	if err != nil {
		return err
	}

	client, err := tfremote.NewClient(c.Args.StorageBackend, config)
	if err != nil {
		return err
	}

	stateBytes, err := ioutil.ReadFile(c.Args.StateFile)
	if err != nil {
		return fmt.Errorf("error reading state file %s: %s", c.Args.StateFile, err)
	}

	err = client.Put(stateBytes)
	if err != nil {
		return fmt.Errorf("error publishing state to %s: %s", c.Args.StorageBackend, err)
	}

	return nil
}
