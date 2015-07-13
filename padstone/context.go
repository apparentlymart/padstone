package padstone

import (
	"fmt"

	tfmodcfg "github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

type Context struct {
	Config         *Config
	State          *terraform.State
	TemporaryState *terraform.State
	ResultState    *terraform.State
	Providers      map[string]terraform.ResourceProviderFactory
	Provisioners   map[string]terraform.ResourceProvisionerFactory
	Variables      map[string]string
	Hooks          []terraform.Hook
	UIInput        terraform.UIInput
	ModuleStorage  tfmodcfg.Storage
}

func (c *Context) Validate() ([]string, []error) {
	tfctx, err := c.newTFContext(c.State, false)
	if err != nil {
		return []string{}, []error{err}
	}
	return tfctx.Validate()
}

func (c *Context) Build() error {
	var err error
	c.State, err = c.planApply(c.State, false)
	c.splitState()
	return err
}

func (c *Context) CleanUp() error {
	var err error
	c.TemporaryState, err = c.planApply(c.TemporaryState, true)
	if err == nil {
		c.removeStateItems(c.State, c.Config.Temporaries)
	}
	return err
}

func (c *Context) Destroy() error {
	var err error
	c.State, err = c.planApply(c.State, true)
	c.splitState()
	return err
}

func (c *Context) splitState() {
	// Turn our single state into separate "temporary" and "result" states,
	// so we can act on them separately in the later parts of the lifecycle.
	c.TemporaryState = c.State.DeepCopy()
	c.ResultState = c.State.DeepCopy()
	c.removeStateItems(c.ResultState, c.Config.Temporaries)
	c.removeStateItems(c.TemporaryState, c.Config.Results)
}

func (c *Context) planApply(state *terraform.State, destroy bool) (*terraform.State, error) {
	tfctx, err := c.newTFContext(state, destroy)
	if err != nil {
		return state, err
	}

	_, err = tfctx.Plan()
	if err != nil {
		return state, fmt.Errorf("error while planning: %s", err)
	}

	newState, err := tfctx.Apply()
	if err != nil {
		return state, fmt.Errorf("error while applying: %s", err)
	}

	return newState, nil
}

func (c *Context) newTFContext(state *terraform.State, destroy bool) (*terraform.Context, error) {
	tfModule := c.Config.TerraformModuleTree()

	err := tfModule.Load(c.ModuleStorage, tfmodcfg.GetModeNone)
	if err != nil {
		return nil, err
	}

	return terraform.NewContext(&terraform.ContextOpts{
		Destroy:      destroy,
		Module:       tfModule,
		State:        state,
		Hooks:        c.Hooks,
		Providers:    c.Providers,
		Provisioners: c.Provisioners,
		Variables:    c.Variables,
		UIInput:      c.UIInput,
	}), nil
}

func (c *Context) removeStateItems(state *terraform.State, resourceSet *ResourceSet) {

	for _, modState := range state.Modules {
		if len(modState.Path) == 1 && modState.Path[0] == "root" {
			// For the root module we will remove the resources we were asked to remove.

			for _, resource := range resourceSet.Resources {
				if _, ok := modState.Resources[resource.Id()]; ok {
					delete(modState.Resources, resource.Id())
				}
			}
		} else {
			// For descendent modules we either keep or remove the entire module
			// depending on whether it's a keeper.
			// FIXME: Need to actually implement this.
		}
	}

}
