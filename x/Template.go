package x

import (
	"embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

func NewTemplate() *Template {
	return &Template{}
}

var (
	TemplateRoot   = "../src/templates"
	TemplateSuffix = ".htm"
	TemplateFuncs  = template.FuncMap{
		"include": func(string) string { return "" },
		"replace": strings.ReplaceAll,
		"jsonEncode": func(v interface{}) template.JS {
			js := JsonEncode(v)
			return template.JS(js)
		},
		"jsonDecode": func(v interface{}) interface{} {
			return JsonDecode(v)
		},
		"jscode": func(v interface{}) template.JS {
			return template.JS(fmt.Sprint(v))
		},
		"htmlcode": func(v interface{}) template.HTML {
			return template.HTML(fmt.Sprint(v))
		},
		"asint": AsInt,
	}
)

var (
	templateUseEmbed  bool
	embedTemplates    embed.FS
	embedTemplatePath string
)

//添加模板函数(全局)
func TemplateAddFuncs(vals ...interface{}) { // {{{
	i := 0
	l := len(vals)
	for i+1 < l {
		TemplateFuncs[vals[i].(string)] = vals[i+1]

		i = i + 2
	}
} // }}}

//设置使用embed.FS
func TemplateEmbed(filesys embed.FS, fs_path string) { // {{{
	templateUseEmbed = true
	embedTemplates = filesys
	embedTemplatePath = fs_path
} // }}}

var (
	templateCache = map[string]*template.Template{}
)

type Template struct {
	vals      map[string]interface{}
	lock      sync.RWMutex
	recursion int
	funcs     template.FuncMap
}

//解析模板变量
func (this *Template) Assign(vals ...interface{}) { // {{{
	if nil == this.vals {
		this.vals = make(map[string]interface{})
	}

	l := len(vals)
	if l == 1 {
		if vals_map, ok := vals[0].(map[string]interface{}); ok {
			for k, v := range vals_map {
				this.vals[k] = v
			}
		}
	} else {
		i := 0
		for i+1 < l {
			this.vals[vals[i].(string)] = vals[i+1]

			i = i + 2
		}
	}
} // }}}

//添加模板函数
func (this *Template) AddFunc(vals ...interface{}) { // {{{
	if nil == this.funcs {
		this.funcs = make(map[string]interface{})
	}

	i := 0
	l := len(vals)
	for i+1 < l {
		this.funcs[vals[i].(string)] = vals[i+1]

		i = i + 2
	}
} // }}}

func (this *Template) Render(w http.ResponseWriter, uri, file string) error { // {{{
	t, inc_info, err := this.loadTemplate(uri, file)
	if err != nil {
		return err
	}

	err = t.Execute(w, this.vals)
	if err != nil {
		return fmt.Errorf("Render template failed, err: %v, include files info[%v]", err, JsonEncode(inc_info))
	}

	return nil
} // }}}

func (this *Template) loadTemplate(uri, template_file string) (*template.Template, interface{}, error) { // {{{
	this.lock.RLock()

	if t, ok := templateCache[uri+"-"+template_file]; ok {
		this.lock.RUnlock()
		return t, nil, nil
	}

	this.lock.RUnlock()

	this.lock.Lock()
	defer this.lock.Unlock()

	data, inc_info, err := this.loadTemplateData(template_file)
	if err != nil {
		return nil, nil, err
	}

	//合并funcmap
	funcs := TemplateFuncs
	for k, v := range this.funcs {
		funcs[k] = v
	}

	t, err := template.New(uri + ":" + template_file).Funcs(funcs).Parse(data)
	if err != nil {
		return nil, nil, fmt.Errorf("Parse template failed, err: %v, include files info[%v]", err, JsonEncode(inc_info))
	}

	templateCache[uri+"-"+template_file] = t

	return t, inc_info, nil
} // }}}

func (this *Template) loadTemplateData(template_file string) (string, interface{}, error) { // {{{
	var buffer []byte
	var err error
	if templateUseEmbed {
		if embedTemplatePath != "" {
			embedTemplatePath = strings.Trim(embedTemplatePath, "/") + "/"
		}
		buffer, err = embedTemplates.ReadFile(embedTemplatePath + template_file)
	} else {
		template_root := Conf.Get("template_root")
		if "" == template_root {
			template_root = TemplateRoot
		}

		if "" != template_root && '/' != template_root[0] && "" != AppRoot {
			template_root = AppRoot + "/" + template_root
		}

		f, err := os.Open(template_root + "/" + template_file)
		if err != nil {
			return "", nil, err
		}
		defer f.Close()

		buffer, err = ioutil.ReadAll(f)
	}

	if err != nil {
		return "", nil, err
	}

	data := string(buffer)
	lines := strings.Count(data, "\n")
	inc := map[string]interface{}{}
	re := regexp.MustCompile("{{\\s*include\\s+(.*?)\\s*}}")
	matches := re.FindAllStringSubmatchIndex(data, -1)
	if len(matches) > 0 {
		new_data := ""
		start := 0
		for _, m := range matches {
			if len(m) == 4 {
				new_data += data[start:m[0]]
				inc_name := strings.Trim(data[m[2]:m[3]], " \r\t\v\n\"'")

				recursion_limit := Conf.GetInt("recursion_limit", 3)
				if this.recursion++; this.recursion > recursion_limit {
					return "", nil, fmt.Errorf("The recursion is too many times, no more than 3 times, you can modify it in the configuration file by [recursion_limit]")
				}

				inc_data, inc_info, err := this.loadTemplateData(inc_name)
				if err != nil {
					return "", nil, fmt.Errorf("Failed to load include file [%s] from [%s], err: %v", inc_name, template_file, err)
					//inc_data = data[m[0]:m[1]]
				}

				inc[inc_name] = inc_info

				start = m[1]
				new_data += inc_data
			}
		}
		if start < len(data) {
			new_data += data[start:]
		}
		data = new_data
	}

	return data, map[string]interface{}{"lines": lines, "inc": inc}, nil
} // }}}
