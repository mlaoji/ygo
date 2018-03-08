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
	jsonData   map[string]map[string]interface{}
}

func (this *Config) Init(configFile string) {
	this.data = make(map[string]map[string]string)
	this.jsonData = make(map[string]map[string]interface{})
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

	if this.jsonData[section] == nil {
		this.jsonData[section] = make(map[string]interface{})
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

			if this.jsonData[section] == nil {
				this.jsonData[section] = make(map[string]interface{})
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			for i, part := range parts {
				parts[i] = strings.Trim(part, emptyRunes)
			}

			key := parts[0]
			value := parts[1]
			//include json file
			if len(value) > 0 && value[0] == '<' && value[len(value)-1] == '>' {
				includes := strings.SplitN(strings.TrimSpace(value[1:len(value)-1]), " ", 2)
				if len(includes) == 2 && strings.EqualFold(includes[0], "include") {
					this.jsonData[section][key], err = this.loadJson(includes[1])
					if nil != err {
						return err
					}
				}
			}
			this.data[section][key] = value
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

func (this *Config) loadJson(file string) (interface{}, error) { // {{{
	confDir := path.Dir(this.configFile)
	newConfName := strings.Trim(file, emptyRunes)
	newConfPath := path.Join(confDir, newConfName)

	stream, err := ioutil.ReadFile(newConfPath)
	if err != nil {
		return nil, errors.New("cannot load json config file:" + err.Error())
	}

	result := JsonDecode(string(stream))
	if result == nil && len(stream) > 0 {
		return nil, errors.New("invalid json format for file:" + file)
	}

	return result, nil
} // }}}

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

func (this *Config) GetJson(keys ...string) interface{} { // {{{
	key, section := this.parseKey(keys...)

	parts := []string{}
	if strings.Contains(key, ".") {
		parts = strings.Split(key, ".")
		key = parts[0]
	}

	if value, ok := this.jsonData[section][key]; ok {
		if len(parts) > 1 {
			for _, v := range parts[1:] {
				if newval, ok := value.(map[string]interface{}); ok {
					value = newval[v]
				}
			}
		}

		return value
	}

	return nil
} // }}}

func (this *Config) GetJsonInt(keys ...string) int { //{{{
	value := this.GetJson(keys...)
	return AsInt(value)
} //}}}

func (this *Config) GetJsonString(keys ...string) string { // {{{
	value := this.GetJson(keys...)
	return AsString(value)
} // }}}

func (this *Config) GetJsonBool(keys ...string) bool { // {{{
	value := this.GetJson(keys...)

	result, err := strconv.ParseBool(AsString(value))
	if err != nil {
		result = false
	}

	return result
} // }}}

func (this *Config) GetJsonSlice(keys ...string) []string { // {{{
	value := this.GetJson(keys...)

	slice := []string{}
	for _, v := range value.([]interface{}) {
		slice = append(slice, AsString(v))
	}

	return slice
} // }}}

func (this *Config) GetJsonSliceInt(keys ...string) []int { // {{{
	value := this.GetJson(keys...)

	slice := []int{}
	for _, v := range value.([]interface{}) {
		slice = append(slice, AsInt(v))
	}

	return slice
} // }}}

func (this *Config) GetJsonSliceMap(keys ...string) []map[string]string { // {{{
	value := this.GetJson(keys...)

	res := []map[string]string{}
	for _, v := range value.([]interface{}) {
		value := map[string]string{}
		for k1, v1 := range v.(map[string]interface{}) {
			value[k1] = AsString(v1)
		}
		res = append(res, value)
	}

	return res
} // }}}

func (this *Config) GetJsonMap(keys ...string) map[string]string { // {{{
	value := this.GetJson(keys...)

	res := map[string]string{}
	for k, v := range value.(map[string]interface{}) {
		res[k] = AsString(v)
	}

	return res
} // }}}

func (this *Config) GetJsonMapInt(keys ...string) map[string]int { // {{{
	value := this.GetJson(keys...)

	res := map[string]int{}
	for k, v := range value.(map[string]interface{}) {
		res[k] = AsInt(v)
	}

	return res
} // }}}
