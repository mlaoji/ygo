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

func (this *Config) Init(configFile string) error {
	this.data = make(map[string]map[string]string)
	this.jsonData = make(map[string]map[string]interface{})
	this.configFile = configFile
	err := this.Load(configFile)
	if err != nil {
		return err
	}

	fmt.Println("Config Init: ", configFile)

	return nil
}

const emptyRunes = " \r\t\v"

func (this *Config) Load(configFile string) error { // {{{
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

	jsonKey := ""
	jsonVal := ""
	injson := false
	for _, line := range lines {
		line = strings.Trim(line, emptyRunes)
		if line == "" || line[0] == '#' || (line[0] == '/' && line[1] == '/') {
			continue
		}

		if injson {
			jsonVal += line

			l := len(line)
			if l > 2 && line[l-3:] == "```" {
				_jsonStr := jsonVal[3 : len(jsonVal)-3]
				_jsonObj := JsonDecode(_jsonStr)
				if _jsonObj == nil && len(_jsonStr) > 0 {
					return errors.New("invalid json format for key:" + jsonKey)
				}
				this.jsonData[section][jsonKey] = _jsonObj

				jsonKey = ""
				jsonVal = ""
				injson = false
			}
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

			//parse json val
			l := len(value)
			if l > 2 && value[0:3] == "```" {
				jsonKey = key
				jsonVal = value
				injson = true

				if l > 5 && value[l-3:] == "```" {
					_jsonStr := jsonVal[3 : len(jsonVal)-3]
					_jsonObj := JsonDecode(_jsonStr)
					if _jsonObj == nil && len(_jsonStr) > 0 {
						return errors.New("invalid json format for key:" + jsonKey)
					}

					this.jsonData[section][jsonKey] = _jsonObj
					jsonKey = ""
					jsonVal = ""
					injson = false
				}
				continue
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
} // }}}

func (this *Config) GetAll(section string) map[string]string {
	return this.data[section]
}

func (this *Config) parseKey(keys ...string) (string, string) { // {{{
	key := keys[0]
	section := "default"
	if len(keys) > 1 {
		section = keys[0]
		key = keys[1]
	}

	return section, key
} // }}}

func (this *Config) Get(keys ...string) string { // {{{
	section, key := this.parseKey(keys...)

	if value, ok := this.data[section][key]; ok {
		return value
	}
	return ""
} // }}}

func (this *Config) GetInt(keys ...string) int { // {{{
	value := this.Get(keys...)
	if value == "" {
		return 0
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return result
} // }}}

func (this *Config) GetBool(keys ...string) bool { // {{{
	value := this.Get(keys...)
	if value == "" {
		return false
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		result = false
	}
	return result
} // }}}

func (this *Config) GetInt64(keys ...string) int64 { // {{{
	value := this.Get(keys...)
	if value == "" {
		return 0
	}
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return result
} // }}}

func (this *Config) GetSlice(keys ...string) []string { // {{{
	separator := ","

	slice := []string{}
	value := this.Get(keys...)
	if value != "" {
		for _, part := range strings.Split(value, separator) {
			slice = append(slice, strings.Trim(part, emptyRunes))
		}
	}
	return slice
} // }}}

func (this *Config) GetSliceInt(keys ...string) []int { // {{{
	slice := this.GetSlice(keys...)
	results := []int{}
	for _, part := range slice {
		result, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		results = append(results, result)
	}
	return results
} // }}}

func (this *Config) GetJson(keys ...string) interface{} { // {{{
	section, key := this.parseKey(keys...)

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
	if newval, ok := value.([]interface{}); ok {
		for _, v := range newval {
			slice = append(slice, AsString(v))
		}
	}

	return slice
} // }}}

func (this *Config) GetJsonSliceInt(keys ...string) []int { // {{{
	value := this.GetJson(keys...)

	slice := []int{}

	if newval, ok := value.([]interface{}); ok {
		for _, v := range newval {
			slice = append(slice, AsInt(v))
		}
	}

	return slice
} // }}}

func (this *Config) GetJsonSliceMap(keys ...string) []map[string]string { // {{{
	value := this.GetJson(keys...)

	res := []map[string]string{}
	if newval, ok := value.([]interface{}); ok {
		for _, v := range newval {
			value := map[string]string{}
			for k1, v1 := range v.(map[string]interface{}) {
				value[k1] = AsString(v1)
			}
			res = append(res, value)
		}
	}

	return res
} // }}}

func (this *Config) GetJsonMap(keys ...string) map[string]string { // {{{
	value := this.GetJson(keys...)

	res := map[string]string{}
	if newval, ok := value.(map[string]interface{}); ok {
		for k, v := range newval {
			res[k] = AsString(v)
		}
	}

	return res
} // }}}

func (this *Config) GetJsonMapInt(keys ...string) map[string]int { // {{{
	value := this.GetJson(keys...)

	res := map[string]int{}

	if newval, ok := value.(map[string]interface{}); ok {
		for k, v := range newval {
			res[k] = AsInt(v)
		}
	}

	return res
} // }}}
