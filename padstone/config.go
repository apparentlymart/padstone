package padstone

import (
	"io/ioutil"

	"github.com/hashicorp/hcl"
	hclhcl "github.com/hashicorp/hcl/hcl"

	tfcfg "github.com/hashicorp/terraform/config"
	tfmodcfg "github.com/hashicorp/terraform/config/module"
)

type Config struct {
	Filename string

	Variables []*tfcfg.Variable
	Artifacts []*Artifact
	Outputs   []*tfcfg.Output
}

type Artifact struct {
	Name string

	Providers    []*tfcfg.ProviderConfig
	Intermediate *ResourceSet
	Result       *ResourceSet
}

type ResourceSet struct {
	Modules   []*tfcfg.Module
	Resources []*tfcfg.Resource
}

func LoadConfig(filename string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseConfig(configBytes, filename)
}

func ParseConfig(configBytes []byte, filename string) (*Config, error) {
	rawConfig, err := hcl.Parse(string(configBytes))
	if err != nil {
		return nil, err
	}

	return NewConfigFromHCL(rawConfig, filename)
}

func NewConfigFromHCL(rawConfig *hclhcl.Object, filename string) (*Config, error) {
	config := &Config{
		Filename: filename,
	}

	if rawVariables := rawConfig.Get("variable", false); rawVariables != nil {
		variables, err := loadVariablesHcl(rawVariables)
		if err != nil {
			return nil, err
		}
		config.Variables = variables
	}

	return config, nil
}

func loadVariablesHcl(rawConfig *hclhcl.Object) ([]*tfcfg.Variable, error) {

	type hclVariable struct {
		Default     interface{}
		Description string
		Fields      []string `hcl:",decodedFields"`
	}

	var variablesHcl map[string]*hclVariable

	err := hcl.DecodeObject(&variablesHcl, rawConfig)
	if err != nil {
		return nil, err
	}

	variables := make([]*tfcfg.Variable, 0, len(variablesHcl))

	for k, v := range variablesHcl {
		if ms, ok := v.Default.([]map[string]interface{}); ok {
			def := make(map[string]interface{})
			for _, m := range ms {
				for k, v := range m {
					def[k] = v
				}
			}
			v.Default = def
		}

		variable := &tfcfg.Variable{
			Name:        k,
			Default:     v.Default,
			Description: v.Description,
		}

		variables = append(variables, variable)
	}

	return variables, nil
}

func (c *Config) TerraformWorkTree() *tfmodcfg.Tree {
	return c.terraformTree(true)
}

func (c *Config) TerraformResultTree() *tfmodcfg.Tree {
	return c.terraformTree(false)
}

func (c *Config) terraformTree(includeIntermediates bool) *tfmodcfg.Tree {

	rootConfig := &tfcfg.Config{
	}

	root := tfmodcfg.NewTree("", rootConfig)

	return root
}
