package x

import (
	"fmt"
	"github.com/mlaoji/ygo/x/yaml"
)

var (
	defaultConfigPath []string
)

func SetConfig(configFiles ...string) {
	defaultConfigPath = configFiles
}

type Config struct {
	*yaml.YamlTree
}

func NewConfig(configFile string) (*Config, string, error) { // {{{
	if configFile != "" {
		if isfile, _ := IsFile(configFile); !isfile {
			configFile = ""
		}
	}

	if configFile == "" {
		for _, v := range defaultConfigPath {
			if isfile, _ := IsFile(v); isfile {
				configFile = v
				break
			}
		}
	}

	if configFile == "" {
		return nil, "", fmt.Errorf("config file is not exists!")
	}

	y, err := yaml.NewYaml(configFile)
	if err != nil {
		return nil, "", err
	}

	return &Config{y.GetYaml()}, configFile, nil
} // }}}
