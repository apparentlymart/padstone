package padstone

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl"
	hclhcl "github.com/hashicorp/hcl/hcl"

	tfcfg "github.com/hashicorp/terraform/config"
	tfmodcfg "github.com/hashicorp/terraform/config/module"
)

type Config struct {
	SourcePath string

	Variables   []*tfcfg.Variable
	Providers   []*tfcfg.ProviderConfig
	Temporaries *ResourceSet
	Results     *ResourceSet
	Outputs     []*tfcfg.Output
}

type ResourceSet struct {
	Modules   []*tfcfg.Module
	Resources []*tfcfg.Resource
}

func LoadConfig(path string) (*Config, error) {
	filenames, err := configFilesInDir(path)
	if err != nil {
		return nil, err
	}

	// We make a separate config for each file below, and then merge
	// them all into this master config for the entire directory.
	config := &Config{
		SourcePath: path,
		Variables:  []*tfcfg.Variable{},
		Temporaries: &ResourceSet{
			Modules:   []*tfcfg.Module{},
			Resources: []*tfcfg.Resource{},
		},
		Results: &ResourceSet{
			Modules:   []*tfcfg.Module{},
			Resources: []*tfcfg.Resource{},
		},
		Providers: []*tfcfg.ProviderConfig{},
		Outputs:   []*tfcfg.Output{},
	}

	for _, filename := range filenames {
		configBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		fileConfig, err := ParseConfig(configBytes, filename)
		if err != nil {
			return nil, err
		}

		// Merge into the aggregate config
		for _, variable := range fileConfig.Variables {
			config.Variables = append(config.Variables, variable)
		}
		for _, resource := range fileConfig.Temporaries.Resources {
			config.Temporaries.Resources = append(
				config.Temporaries.Resources, resource,
			)
		}
		for _, module := range fileConfig.Temporaries.Modules {
			config.Temporaries.Modules = append(
				config.Temporaries.Modules, module,
			)
		}
		for _, resource := range fileConfig.Results.Resources {
			config.Results.Resources = append(
				config.Results.Resources, resource,
			)
		}
		for _, module := range fileConfig.Results.Modules {
			config.Results.Modules = append(
				config.Results.Modules, module,
			)
		}
		for _, provider := range fileConfig.Providers {
			config.Providers = append(config.Providers, provider)
		}
		for _, output := range fileConfig.Outputs {
			config.Outputs = append(config.Outputs, output)
		}
	}

	return config, nil
}

func ParseConfig(configBytes []byte, filename string) (*Config, error) {
	rawConfig, err := hcl.Parse(string(configBytes))
	if err != nil {
		return nil, err
	}

	return NewConfigFromHCL(rawConfig, filename)
}

func NewConfigFromHCL(rawConfig *hclhcl.Object, filename string) (*Config, error) {
	fmt.Print("\n")
	config := &Config{
		SourcePath: filename,
		Temporaries: &ResourceSet{
			Resources: []*tfcfg.Resource{},
			Modules:   []*tfcfg.Module{},
		},
		Results: &ResourceSet{
			Resources: []*tfcfg.Resource{},
			Modules:   []*tfcfg.Module{},
		},
	}

	var err error

	if rawVariables := rawConfig.Get("variable", false); rawVariables != nil {
		variables, err := loadVariablesHcl(rawVariables)
		if err != nil {
			return nil, err
		}
		config.Variables = variables
	}

	if rawResources := rawConfig.Get("temporary_resource", false); rawResources != nil {
		config.Temporaries.Resources, err = tfcfg.LoadResourcesHCL(rawResources)
		if err != nil {
			return nil, err
		}
	}

	if rawModules := rawConfig.Get("temporary_module", false); rawModules != nil {
		config.Temporaries.Modules, err = tfcfg.LoadModulesHCL(rawModules)
		if err != nil {
			return nil, err
		}
	}

	if rawResources := rawConfig.Get("resource", false); rawResources != nil {
		config.Results.Resources, err = tfcfg.LoadResourcesHCL(rawResources)
		if err != nil {
			return nil, err
		}
	}

	if rawModules := rawConfig.Get("module", false); rawModules != nil {
		config.Results.Modules, err = tfcfg.LoadModulesHCL(rawModules)
		if err != nil {
			return nil, err
		}
	}

	if rawOutputs := rawConfig.Get("output", false); rawOutputs != nil {
		var err error
		config.Outputs, err = tfcfg.LoadOutputsHCL(rawOutputs)
		if err != nil {
			return nil, err
		}
	} else {
		config.Outputs = []*tfcfg.Output{}
	}

	if rawProviders := rawConfig.Get("provider", false); rawProviders != nil {
		var err error
		config.Providers, err = tfcfg.LoadProvidersHCL(rawProviders)
		if err != nil {
			return nil, err
		}
	} else {
		config.Providers = []*tfcfg.ProviderConfig{}
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

func configFilesInDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, fmt.Errorf(
			"configuration path is a file, but we need a directory: %s",
			dir,
		)
	}

	var files []string
	err = nil

	for err != io.EOF {
		var fis []os.FileInfo
		fis, err = f.Readdir(128)
		if err != nil && err != io.EOF {
			return nil, err
		}

		for _, fi := range fis {
			if fi.IsDir() {
				continue
			}

			name := fi.Name()
			extValue := ext(name)
			if extValue == "" || isIgnoredFile(name) {
				continue
			}

			path := filepath.Join(dir, name)
			files = append(files, path)
		}
	}

	return files, nil
}

func (c *Config) TerraformModuleTree() *tfmodcfg.Tree {
	tfConfig := &tfcfg.Config{
		Dir:       c.SourcePath,
		Variables: c.Variables,
		Resources: make(
			[]*tfcfg.Resource,
			0,
			len(c.Temporaries.Resources)+len(c.Results.Resources),
		),
		Modules: make(
			[]*tfcfg.Module,
			0,
			len(c.Temporaries.Modules)+len(c.Results.Modules),
		),
	}

	for _, resource := range c.Temporaries.Resources {
		tfConfig.Resources = append(tfConfig.Resources, resource)
	}
	for _, module := range c.Temporaries.Modules {
		tfConfig.Modules = append(tfConfig.Modules, module)
	}

	for _, resource := range c.Results.Resources {
		tfConfig.Resources = append(tfConfig.Resources, resource)
	}
	for _, module := range c.Results.Modules {
		tfConfig.Modules = append(tfConfig.Modules, module)
	}

	return tfmodcfg.NewTree("", tfConfig)
}

func ext(path string) string {
	if strings.HasSuffix(path, ".pad") {
		return ".pad"
	} else if strings.HasSuffix(path, ".pad.json") {
		return ".pad.json"
	} else {
		return ""
	}
}

func isIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		(strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#")) // emacs
}
