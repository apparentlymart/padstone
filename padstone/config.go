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

	Variables []*tfcfg.Variable
	Providers []*tfcfg.ProviderConfig
	Artifacts []*Artifact
	Outputs   []*tfcfg.Output
}

type Artifact struct {
	Name string

	Intermediates *ResourceSet
	Results       *ResourceSet
	Outputs       []*tfcfg.Output
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
		Artifacts:  []*Artifact{},
		Providers:  []*tfcfg.ProviderConfig{},
		Outputs:    []*tfcfg.Output{},
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
		for _, artifact := range fileConfig.Artifacts {
			config.Artifacts = append(config.Artifacts, artifact)
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
	}

	if rawVariables := rawConfig.Get("variable", false); rawVariables != nil {
		variables, err := loadVariablesHcl(rawVariables)
		if err != nil {
			return nil, err
		}
		config.Variables = variables
	}

	if rawArtifacts := rawConfig.Get("artifact", false); rawArtifacts != nil {
		artifacts, err := loadArtifactsHcl(rawArtifacts)
		if err != nil {
			return nil, err
		}
		config.Artifacts = artifacts
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

func loadArtifactsHcl(rawConfig *hclhcl.Object) ([]*Artifact, error) {
	artifactsHcl := rawConfig.Elem(true)

	artifacts := make([]*Artifact, 0, len(artifactsHcl))

	type artifactHclStruct struct {
		Intermediates *hclhcl.Object
		Results       *hclhcl.Object
	}

	var err error
	for _, v := range artifactsHcl {
		artifact := &Artifact{
			Name: v.Key,
		}

		if rawIntermediates := v.Get("intermediates", false); rawIntermediates != nil {
			artifact.Intermediates, err = loadResourceSetHcl(rawIntermediates)
			if err != nil {
				return nil, err
			}
		} else {
			artifact.Intermediates = emptyResourceSet()
		}

		if rawResults := v.Get("results", false); rawResults != nil {
			artifact.Results, err = loadResourceSetHcl(rawResults)
			if err != nil {
				return nil, err
			}
		} else {
			artifact.Results = emptyResourceSet()
		}

		if rawOutputs := v.Get("output", false); rawOutputs != nil {
			artifact.Outputs, err = tfcfg.LoadOutputsHCL(rawOutputs)
			if err != nil {
				return nil, err
			}
		} else {
			artifact.Outputs = []*tfcfg.Output{}
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func loadResourceSetHcl(rawConfig *hclhcl.Object) (*ResourceSet, error) {
	set := &ResourceSet{
		Modules:   []*tfcfg.Module{},
		Resources: []*tfcfg.Resource{},
	}

	var err error

	if rawModules := rawConfig.Get("module", false); rawModules != nil {
		set.Modules, err = tfcfg.LoadModulesHCL(rawModules)
		if err != nil {
			return nil, err
		}
	}

	if rawResources := rawConfig.Get("resource", false); rawResources != nil {
		set.Resources, err = tfcfg.LoadResourcesHCL(rawResources)
		if err != nil {
			return nil, err
		}
	}

	return set, nil
}

func emptyResourceSet() *ResourceSet {
	return &ResourceSet{
		Modules:   []*tfcfg.Module{},
		Resources: []*tfcfg.Resource{},
	}
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

func (c *Config) TerraformWorkTree() *tfmodcfg.Tree {
	return c.terraformTree(true)
}

func (c *Config) TerraformResultTree() *tfmodcfg.Tree {
	return c.terraformTree(false)
}

func (c *Config) terraformTree(includeIntermediates bool) *tfmodcfg.Tree {

	rootConfig := &tfcfg.Config{
		Variables: c.Variables,
	}

	root := tfmodcfg.NewTree("", rootConfig)
	root.SetPath([]string{"root"})

	return root
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
