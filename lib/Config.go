package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
)

var Conf = &Config{}

type Config struct {
	configFile string
	data       map[string]map[string]string
}

func (this *Config) Init(configFile string) {
	this.data = make(map[string]map[string]string)
	this.configFile = configFile
	err := this.Load(configFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("Config Init: ", configFile)
}

const emptyRunes = " \r\t\v"

func (this *Config) Load(configFile string) error {
	stream, err := ioutil.ReadFile(configFile)
	if err != nil {
		return errors.New("cannot load config file")
	}
	content := string(stream)
	lines := strings.Split(content, "\n")

	section := "default"
	if this.data[section] == nil {
		this.data[section] = make(map[string]string)
	}

	for _, line := range lines {
		line = strings.Trim(line, emptyRunes)
		if line == "" || line[0] == '#' {
			continue
		}

		if line[0] == '[' && line[len(line)-1] == ']' {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if this.data[section] == nil {
				this.data[section] = make(map[string]string)
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			for i, part := range parts {
				parts[i] = strings.Trim(part, emptyRunes)
			}
			this.data[section][parts[0]] = parts[1]
		} else {
			//处理include
			includes := strings.SplitN(parts[0], " ", 2)
			if len(includes) == 2 && strings.EqualFold(includes[0], "include") {
				confDir := path.Dir(configFile)
				newConfName := strings.Trim(includes[1], emptyRunes)
				newConfPath := path.Join(confDir, newConfName)
				err := this.Load(newConfPath)
				if err != nil {
					return errors.New("load include config file failed")
				}
				continue
			} else {
				return errors.New("invalid config file syntax")
			}
		}
	}
	return nil
}

func (this *Config) GetAll(section string) map[string]string {
	return this.data[section]
}

func (this *Config) parseKey(keys ...string) (string, string) {
	key := keys[0]
	section := "default"
	if len(keys) > 1 {
		section = keys[1]
	}

	return key, section
}

func (this *Config) Get(keys ...string) string {
	key, section := this.parseKey(keys...)

	if value, ok := this.data[section][key]; ok {
		return value
	}
	return ""
}

func (this *Config) GetInt(keys ...string) int {
	key, section := this.parseKey(keys...)

	value := this.Get(key, section)
	if value == "" {
		return 0
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return result
}

func (this *Config) GetBool(keys ...string) bool {
	key, section := this.parseKey(keys...)

	value := this.Get(key, section)
	if value == "" {
		return false
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		result = false
	}
	return result
}

func (this *Config) GetInt64(keys ...string) int64 {
	key, section := this.parseKey(keys...)

	value := this.Get(key, section)
	if value == "" {
		return 0
	}
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return result
}

func (this *Config) GetSlice(key string, separators ...string) []string {
	separator := ","
	section := "default"

	if len(separators) > 0 {
		separator, section = this.parseKey(separators...)
	}

	slice := []string{}
	value := this.Get(key, section)
	if value != "" {
		for _, part := range strings.Split(value, separator) {
			slice = append(slice, strings.Trim(part, emptyRunes))
		}
	}
	return slice
}

func (this *Config) GetSliceInt(key string, separators ...string) []int {
	slice := this.GetSlice(key, separators...)
	results := []int{}
	for _, part := range slice {
		result, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		results = append(results, result)
	}
	return results
}

func (this *Config) GetJson(keys ...string) interface{} {
	key, section := this.parseKey(keys...)

	value := this.Get(key, section)
	if value == "" {
		return nil
	}

	confDir := path.Dir(this.configFile)
	newConfName := strings.Trim(value, emptyRunes)
	newConfPath := path.Join(confDir, newConfName)

	stream, err := ioutil.ReadFile(newConfPath)
	if err != nil {
		return errors.New("cannot load json config file for key:" + key + err.Error())
	}

	result := JsonDecode(string(stream))
	if result == nil && len(stream) > 0 {
		return errors.New("invalid json format for key:" + key)
	}

	return result
}
