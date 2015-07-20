package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/terraform"
)

func WriteState(state *terraform.State, outFile string) error {
	// TODO: Make this file location customizable, and also don't re-open this
	// file every time.
	f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening state file: %s\n", err)
	}
	err = terraform.WriteState(state, f)
	if err != nil {
		return fmt.Errorf("error writing state: %s\n", err)
	}
	return nil
}
