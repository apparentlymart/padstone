package padstone

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"

	tfcfg "github.com/hashicorp/terraform/config"
)

type Config struct {
	// SourceFilename is the path to the file from which this configuration
	// was loaded. If it is blank, this configuration wasn't loaded from
	// an on-disk file.
	SourceFilename string

	Variables []*tfcfg.Variable
	Targets   []*TargetConfig
	Providers []*tfcfg.ProviderConfig
}

type TargetConfig struct {
	Name string

	Modules   []*tfcfg.Module
	Providers []*tfcfg.ProviderConfig
	Resources []*tfcfg.Resource
	Outputs   []*tfcfg.Output
}

func LoadConfig(filename string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseConfig(configBytes, filename)
}

func ParseConfig(configBytes []byte, filename string) (*Config, error) {
	rawConfigFile, err := hcl.Parse(string(configBytes))
	if err != nil {
		return nil, err
	}

	rawConfig := rawConfigFile.Node
	return NewConfigFromHCL(rawConfig.(*ast.ObjectList), filename)
}

func NewConfigFromHCL(hclConfig *ast.ObjectList, filename string) (*Config, error) {
	config := &Config{
		SourceFilename: filename,
	}

	var err error

	config.Variables, err = loadConfigVariables(hclConfig.Filter("variable"))
	if err != nil {
		return nil, err
	}

	config.Providers, err = loadConfigProviders(hclConfig.Filter("provider"))
	if err != nil {
		return nil, err
	}

	config.Targets, err = loadConfigTargets(hclConfig.Filter("target"))
	if err != nil {
		return nil, err
	}

	return config, nil
}

func loadConfigVariables(hclConfig *ast.ObjectList) ([]*tfcfg.Variable, error) {
	hclConfig = hclConfig.Children()
	result := make([]*tfcfg.Variable, 0, len(hclConfig.Items))

	if len(hclConfig.Items) == 0 {
		return result, nil
	}

	for _, item := range hclConfig.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("variable '%s': should be a block", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		variable := &tfcfg.Variable{
			Name: n,
		}
		if a := listVal.Filter("default"); len(a.Items) > 0 {
			err := hcl.DecodeObject(&variable.Default, a.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading variable %s default: %s", n, err,
				)
			}
		}
		if a := listVal.Filter("description"); len(a.Items) > 0 {
			err := hcl.DecodeObject(&variable.Description, a.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading variable %s description: %s", n, err,
				)
			}
		}
		if a := listVal.Filter("type"); len(a.Items) > 0 {
			err := hcl.DecodeObject(&variable.DeclaredType, a.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading variable %s type: %s", n, err,
				)
			}
		}

		result = append(result, variable)
	}

	return result, nil
}

func loadConfigProviders(hclConfig *ast.ObjectList) ([]*tfcfg.ProviderConfig, error) {
	hclConfig = hclConfig.Children()
	result := make([]*tfcfg.ProviderConfig, 0, len(hclConfig.Items))

	if len(hclConfig.Items) == 0 {
		return result, nil
	}

	for _, item := range hclConfig.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("provider '%s': should be a block", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		delete(config, "alias")

		rawConfig, err := tfcfg.NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"error reading provider config %s: %s", n, err,
			)
		}

		// If we have an alias, add it in
		var alias string
		if a := listVal.Filter("alias"); len(a.Items) > 0 {
			err := hcl.DecodeObject(&alias, a.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading provider %s alias: %s", n, err,
				)
			}
		}

		result = append(result, &tfcfg.ProviderConfig{
			Name:      n,
			Alias:     alias,
			RawConfig: rawConfig,
		})
	}

	return result, nil
}

func loadConfigTargets(hclConfig *ast.ObjectList) ([]*TargetConfig, error) {
	hclConfig = hclConfig.Children()
	result := make([]*TargetConfig, 0, len(hclConfig.Items))

	if len(hclConfig.Items) == 0 {
		return result, nil
	}

	for _, item := range hclConfig.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("target '%s': should be a block", n)
		}

		target := &TargetConfig{
			Name: n,
		}

		var err error

		target.Providers, err = loadConfigProviders(listVal.Filter("provider"))
		if err != nil {
			return nil, err
		}

		target.Modules, err = loadConfigModules(listVal.Filter("module"))
		if err != nil {
			return nil, err
		}

		target.Outputs, err = loadConfigOutputs(listVal.Filter("output"))
		if err != nil {
			return nil, err
		}

		result = append(result, target)
	}

	return result, nil
}

func loadConfigModules(hclConfig *ast.ObjectList) ([]*tfcfg.Module, error) {
	hclConfig = hclConfig.Children()
	result := make([]*tfcfg.Module, 0, len(hclConfig.Items))

	if len(hclConfig.Items) == 0 {
		return result, nil
	}

	for _, item := range hclConfig.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("module '%s': should be a block", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		delete(config, "source")

		rawConfig, err := tfcfg.NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"error reading module config %s: %s", n, err,
			)
		}

		var source string
		if a := listVal.Filter("source"); len(a.Items) > 0 {
			err := hcl.DecodeObject(&source, a.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading module %s source: %s", n, err,
				)
			}
		}

		result = append(result, &tfcfg.Module{
			Name:      n,
			Source:    source,
			RawConfig: rawConfig,
		})
	}

	return result, nil
}

func loadConfigOutputs(hclConfig *ast.ObjectList) ([]*tfcfg.Output, error) {
	hclConfig = hclConfig.Children()
	result := make([]*tfcfg.Output, 0, len(hclConfig.Items))

	if len(hclConfig.Items) == 0 {
		return result, nil
	}

	for _, item := range hclConfig.Items {
		n := item.Keys[0].Token.Value().(string)

		if _, ok := item.Val.(*ast.ObjectType); !ok {
			return nil, fmt.Errorf("output '%s': should be a block", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		rawConfig, err := tfcfg.NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"error reading output config %s: %s", n, err,
			)
		}

		result = append(result, &tfcfg.Output{
			Name:      n,
			RawConfig: rawConfig,
		})
	}

	return result, nil
}
