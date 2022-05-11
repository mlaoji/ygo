package yaml

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

type Yaml struct {
	data []*YamlTree
	file string
}

type YamlTree struct {
	node YamlNode
	sep  string //分隔符
}

func NewYaml(file ...string) (*Yaml, error) { // {{{
	yaml := &Yaml{}

	if len(file) > 0 {
		err := yaml.LoadFile(file[0])
		if err != nil {
			return nil, err
		}
	}

	return yaml, nil
} // }}}

func (this *Yaml) LoadFile(file string) error { // {{{

	this.file = file

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	return this.LoadStream(r)
} // }}}

func (this *Yaml) LoadString(str string) error { // {{{

	r := strings.NewReader(str)
	return this.LoadStream(r)
} // }}}

func (this *Yaml) LoadStream(r io.Reader) error { // {{{

	lb := &lineBuffer{
		Reader: bufio.NewReader(r),
	}

	var err error
	var node YamlNode
	var remain bool

	for {
		node, remain, err = this.parseNode(lb, 0, nil)
		if err != nil {
			return err
		}

		this.data = append(this.data, &YamlTree{node: node})

		if !remain {
			break
		}
	}

	return nil
} // }}}

//获取所有yaml对象
func (this *Yaml) GetYamls() []*YamlTree { // {{{
	return this.data
} // }}}

//获取合并后的yaml对象，同一文档中的多个yaml对象会由下向上合并覆盖相同的key
func (this *Yaml) GetYaml() *YamlTree { // {{{
	return mergeYamlTree(this.data...)
} // }}}

//当key值中含有点(.)时，使用指定的寻址分隔符替代
func (this *YamlTree) UseSep(key string) *YamlTree { // {{{
	return &YamlTree{node: this.node, sep: key}
} // }}}

func (this *YamlTree) GetTree(key string) *YamlTree { // {{{
	return &YamlTree{node: this.get(this.node, key, false), sep: this.sep}
} // }}}

func (this *YamlTree) GetNode(key string) YamlNode { // {{{
	return this.get(this.node, key, false)
} // }}}

func (this *YamlTree) Get(key string, defs ...string) string { // {{{
	def := ""
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)

	if s, ok := val.(YamlScalar); ok {
		return string(s)
	}

	return def
} // }}}

func (this *YamlTree) GetInt(key string, defs ...int) int { // {{{
	def := 0
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlScalar); ok {
		result, err := strconv.Atoi(string(s))
		if err != nil {
			return def
		}

		return result
	}

	return def
} // }}}

func (this *YamlTree) GetBool(key string, defs ...bool) bool { // {{{
	def := false
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlScalar); ok {
		result, err := strconv.ParseBool(string(s))
		if err != nil {
			return def
		}

		return result
	}

	return def
} // }}}

func (this *YamlTree) GetSlice(key string, defs ...[]string) []string { // {{{
	def := []string{}
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlList); ok {
		result := []string{}
		for _, v := range s {
			if j, ok := v.(YamlScalar); ok {
				result = append(result, string(j))
			} else {
				result = append(result, "")
			}
		}

		return result
	}

	return def
} // }}}

func (this *YamlTree) GetSliceInt(key string, defs ...[]int) []int { // {{{
	def := []int{}
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlList); ok {
		result := []int{}
		for _, v := range s {
			if j, ok := v.(YamlScalar); ok {
				i, _ := strconv.Atoi(string(j))
				result = append(result, i)
			} else {
				result = append(result, 0)
			}
		}

		return result
	}

	return def
} // }}}

func (this *YamlTree) GetSliceTree(key string) []*YamlTree { // {{{
	if len(key) == 0 {
		return nil
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlList); ok {
		result := []*YamlTree{}
		for _, v := range s {
			result = append(result, &YamlTree{node: v})
		}

		return result
	}

	return nil
} // }}}

func (this *YamlTree) GetSliceNode(key string) []YamlNode { // {{{
	if len(key) == 0 {
		return nil
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlList); ok {
		result := []YamlNode{}
		for _, v := range s {
			result = append(result, v)
		}

		return result
	}

	return nil
} // }}}

func (this *YamlTree) GetMap(key string, defs ...map[string]string) map[string]string { // {{{
	def := map[string]string{}
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlMap); ok {
		result := map[string]string{}
		for k, v := range s {
			if p, ok := v.(YamlScalar); ok {
				//x, _ := strconv.Atoi(string(p))
				result[k] = string(p)
			} else {
				result[k] = ""
			}
		}

		return result

	}

	return def
} // }}}

func (this *YamlTree) GetMapInt(key string, defs ...map[string]int) map[string]int { // {{{
	def := map[string]int{}
	if len(defs) > 0 {
		def = defs[0]
	}

	if len(key) == 0 {
		return def
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlMap); ok {
		result := map[string]int{}
		for k, v := range s {
			if p, ok := v.(YamlScalar); ok {
				x, _ := strconv.Atoi(string(p))
				result[k] = x
			} else {
				result[k] = 0
			}
		}

		return result

	}

	return def
} // }}}

func (this *YamlTree) GetMapTree(key string) map[string]*YamlTree { // {{{
	if len(key) == 0 {
		return nil
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlMap); ok {
		result := map[string]*YamlTree{}
		for k, v := range s {
			result[k] = &YamlTree{node: v}
		}

		return result

	}

	return nil
} // }}}

func (this *YamlTree) GetMapNode(key string) map[string]YamlNode { // {{{
	if len(key) == 0 {
		return nil
	}

	val := this.get(this.node, key, false)
	if s, ok := val.(YamlMap); ok {
		result := map[string]YamlNode{}
		for k, v := range s {
			result[k] = v
		}

		return result

	}

	return nil
} // }}}

func (this *YamlTree) GetJson(key string) string { // {{{
	val := this.get(this.node, key, false)
	content, err := json.MarshalIndent(val, "", "")
	if err != nil {
		return ""
	}

	return strings.Replace(string(content), "\n", "", -1)
} // }}}

//将yamlTree对象导出为yaml字符串
func (this *YamlTree) ExportYaml() string { // {{{
	return this.exportYaml(this.node, "", true)
} // }}}

func (this *YamlTree) exportYaml(node YamlNode, padding string, inmap bool) string { // {{{
	yaml := ""

	switch val := node.(type) {
	case YamlList:
		if inmap {
			yaml += "\n"
		}

		for k, v := range val {
			p := padding
			if k == 0 && !inmap {
				p = ""
			}
			yaml += p + `- ` + this.exportYaml(v, padding+"  ", false)
		}
	case YamlMap:
		if inmap {
			yaml += "\n"
		}

		first := true
		for k, v := range val {
			p := padding
			if first && !inmap {
				first = false
				p = ""
			}
			yaml += p + k + ": " + this.exportYaml(v, padding+"  ", true)
		}
	case YamlScalar:
		yaml += string(val) + "\n"
	}

	return yaml
} // }}}

//多级联合寻址
func (this *YamlTree) get(node YamlNode, spec string, isList bool) YamlNode { // {{{

	if len(spec) == 0 {
		return node
	}

	if node == nil {
		return nil
	}

	if this.sep == "" {
		this.sep = "."
	}

	sep := strings.IndexAny(spec, this.sep+"[")
	var token, remain string
	nextIsList := false

	if sep < 0 {
		token = spec
		remain = ""
	} else {
		token = spec[:sep]
		remain = spec[(sep + 1):]

		if spec[sep] == '[' {
			nextIsList = true
		}
	}

	if isList {
		s, ok := node.(YamlList)
		if !ok {
			return nil
		}

		if token[len(token)-1] == ']' {
			if num, err := strconv.Atoi(token[:len(token)-1]); err == nil {
				if num >= 0 && num < len(token) {
					return this.get(s[num], remain, nextIsList)
				}
			}
		}
		return nil
	} else {
		m, ok := node.(YamlMap)
		if !ok {
			return nil
		}

		n, ok := m[token]
		if !ok {
			return nil
		}

		return this.get(n, remain, nextIsList)
	}
} // }}}

//////
// Supporting types
const (
	typUnknown = iota
	typSequence
	typMapping
	typScalar
	typInclude
	typJson
)

func (this *Yaml) parseNode(r *lineBuffer, ind int, initial YamlNode) (node YamlNode, remain bool, err error) { // {{{
	first := true
	node = initial

	// read lines
	for {
		line := r.Next(ind)
		if line == nil {
			break
		}

		if line.indent == 0 && string(line.con) == "---" {
			remain = true
			break
		}

		if len(line.con) == 0 {
			continue
		}

		if first {
			ind = line.indent
			first = false
		}

		types := []int{}
		pieces := []string{}

		var inlineValue func([]byte) error
		inlineValue = func(partial []byte) error { // {{{
			// TODO(kevlar): This can be a for loop now
			vtyp, brk := getType(partial)
			begin, end := partial[:brk], partial[brk:]

			if vtyp == typMapping {
				end = end[1:]
			}
			end = bytes.Trim(end, " ")

			switch vtyp {
			case typInclude:
				incPath := path.Join(path.Dir(this.file), string(end))
				err := this.LoadFile(incPath)
				if nil != err {
					return err
				}
				//node = mergeMapNode(node, inc)
				types = append(types, typInclude)
				pieces = append(pieces, "inc")
				return nil
			case typJson:
				types = append(types, typJson)
				pieces = append(pieces, string(end))
			case typScalar:
				l := len(end)
				if l > 0 {
					//去掉字符串的引号
					if end[0] == '"' && end[l-1] == '"' {
						end = bytes.ReplaceAll(end[1:l-1], []byte("\"\""), []byte("\""))
					} else if end[0] == '\'' && end[l-1] == '\'' {
						end = bytes.ReplaceAll(end[1:l-1], []byte("''"), []byte("'"))
					}
				}

				types = append(types, typScalar)
				pieces = append(pieces, string(end))

				return nil
			case typMapping:
				types = append(types, typMapping)
				pieces = append(pieces, string(bytes.TrimSpace(begin)))

				if len(end) == 1 && end[0] == '|' {
					text := ""

					for {
						l := r.Next(1)
						if l == nil {
							break
						}

						s := string(bytes.Trim(l.con, " "))
						if len(s) == 0 {
							break
						}
						text = text + "\n" + s
					}

					types = append(types, typScalar)
					pieces = append(pieces, string(text))
					return nil
				}

				inlineValue(end)
			case typSequence:
				types = append(types, typSequence)
				pieces = append(pieces, "-")

				inlineValue(end)
			}

			return nil
		} // }}}

		err = inlineValue(line.con)
		if err != nil {
			return
		}

		var prev YamlNode
		var prevIsJson bool

		// Nest inlines
		for len(types) > 0 {
			last := len(types) - 1
			typ, piece := types[last], pieces[last]

			var current YamlNode
			if last == 0 {
				current = node
			}
			//child := parseNode(r, line.indent+1, typUnknown) // TODO allow scalar only

			// Add to current node
			switch typ {
			case typScalar: // last will be == nil
				if _, ok := current.(YamlScalar); current != nil && !ok {
					panic("cannot append scalar to non-scalar node => " + piece)
				}
				if current != nil {
					current = YamlScalar(piece) + " " + current.(YamlScalar)
					break
				}
				current = YamlScalar(piece)
			case typMapping:
				var mapNode YamlMap
				var ok bool
				var child YamlNode
				var err error

				// Get the current map, if there is one
				if mapNode, ok = current.(YamlMap); current != nil && !ok {
					_ = current.(YamlMap) // panic
				} else if current == nil {
					mapNode = make(YamlMap)
				}

				if _, inlineMap := prev.(YamlScalar); inlineMap && last > 0 {
					current = YamlMap{
						piece: prev,
					}
					break
				}

				if prevIsJson {
					prevIsJson = false
					child = prev
				} else {
					child, _, err = this.parseNode(r, line.indent+1, prev)
					if err != nil {
						return nil, false, err
					}
				}

				mapNode[piece] = child
				current = mapNode

			case typSequence:
				var listNode YamlList
				var ok bool
				var child YamlNode
				var err error

				// Get the current list, if there is one
				if listNode, ok = current.(YamlList); current != nil && !ok {
					_ = current.(YamlList) // panic
				} else if current == nil {
					listNode = make(YamlList, 0)
				}

				if _, inlineList := prev.(YamlScalar); inlineList && last > 0 {
					current = YamlList{
						prev,
					}
					break
				}

				if prevIsJson {
					prevIsJson = false
					child = prev
				} else {
					child, _, err = this.parseNode(r, line.indent+1, prev)
					if err != nil {
						return nil, false, err
					}
				}

				listNode = append(listNode, child)
				current = listNode

			case typJson:
				current = strToNode(piece)
				prevIsJson = true
			}

			if last < 0 {
				last = 0
			}
			types = types[:last]
			pieces = pieces[:last]
			prev = current
		}

		node = prev
	}
	return
} // }}}

func getType(line []byte) (typ, split int) { // {{{
	l := len(line)
	if l == 0 {
		return
	}

	if l > 8 && string(line[0:8]) == "include " && strings.TrimSpace(string(line[8:])) != "" {
		typ = typInclude
		split = 7
		return
	}

	if line[0] == '-' && (l == 1 || line[1] == ' ') {
		typ = typSequence
		split = 1
		return
	}

	if line[0] == '[' {
		if line[l-1] != ']' {
			panic("unclosed tag")
		}
		typ = typJson
		return
	}

	if line[0] == '{' {
		if line[l-1] != '}' {
			panic("unclosed tag")
		}
		typ = typJson
		return
	}

	typ = typScalar

	if line[0] == '"' || line[0] == '\'' {
		return
	}

	// the first character is real
	// need to iterate past the first word
	// things like "foo:" and "foo :" are mappings
	// everything else is a scalar

	idx := bytes.IndexAny(line, " \":")
	if idx < 0 {
		return
	}

	if line[idx] == '"' {
		return
	}

	if line[idx] == ':' {
		typ = typMapping
		split = idx
	} else if line[idx] == ' ' {
		// we have a space
		// need to see if its all spaces until a :
		for i := idx; i < len(line); i++ {
			switch ch := line[i]; ch {
			case ' ':
				continue
			case ':':
				// only split on colons followed by a space
				if i+1 < len(line) && line[i+1] != ' ' {
					continue
				}

				typ = typMapping
				split = i
				break
			default:
				break
			}
		}
	}

	if typ == typMapping && split+1 < len(line) && line[split+1] != ' ' {
		typ = typScalar
		split = 0
	}

	return
} // }}}

type line struct {
	indent int
	con    []byte
}

type lineBuffer struct {
	*bufio.Reader
	pending *line
}

func (lb *lineBuffer) Next(min int) (next *line) { // {{{
	if lb.pending == nil {
		var (
			read []byte
			more bool
			err  error
		)

		l := new(line)
		more = true
		for more {
			read, more, err = lb.ReadLine()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				panic(err)
			}
			l.con = append(l.con, read...)
		}

		for k, ch := range l.con {
			switch ch {
			case ' ':
				l.indent += 1
				continue
			case '-':
				if len(l.con) == k+1 || l.con[k+1] == ' ' {
					l.indent += 1
				}
			default:
			}
			break
		}
		l.con = bytes.Trim(l.con, " ")

		// Ignore blank lines and comments.
		if len(l.con) == 0 || l.con[0] == '#' {
			return lb.Next(min)
		}

		//trim comments
		idx := bytes.Index(l.con, []byte(" #"))
		if idx > 0 {
			l.con = l.con[:idx]
		}
		//redo
		l.con = bytes.Trim(l.con, " ")

		lb.pending = l
	}
	next = lb.pending

	if next.indent < min {
		return nil
	}
	lb.pending = nil
	return
} // }}}

// A YamlNode is a YAML Node which can be a YamlMap, YamlList or YamlScalar.
type YamlNode interface {
	ToYamlTree() *YamlTree
}

type YamlMap map[string]YamlNode
type YamlList []YamlNode
type YamlScalar string

func (node YamlMap) ToYamlTree() *YamlTree {
	return &YamlTree{node: node}
}

func (node YamlList) ToYamlTree() *YamlTree {
	return &YamlTree{node: node}
}

func (node YamlScalar) ToYamlTree() *YamlTree {
	return &YamlTree{node: node}
}

func mergeYamlTree(trees ...*YamlTree) *YamlTree { // {{{
	var node YamlMap
	for k, v := range trees {
		n, ok := v.node.(YamlMap)
		if !ok {
			n = YamlMap{}
		}

		if k == 0 {
			node = n
		} else {
			for i, j := range n {
				node[i] = j
			}
		}
	}

	return &YamlTree{node: node, sep: "."}
} // }}}

func mergeMapNode(nodes ...YamlNode) YamlNode { // {{{
	var node YamlMap
	for k, v := range nodes {
		n, ok := v.(YamlMap)
		if !ok {
			n = YamlMap{}
		}

		if k == 0 {
			node = n
		} else {
			for i, j := range n {
				node[i] = j
			}
		}
	}

	return YamlNode(node)
} // }}}

//格式化json类型的值
func strToNode(r string) YamlNode { // {{{
	r = fmtJson(r)

	var obj interface{}
	err := json.Unmarshal([]byte(r), &obj)
	if nil != err {
		panic(err)
	}

	return toNode(obj)
} // }}}

func toNode(s interface{}) YamlNode { // {{{
	switch val := s.(type) {
	case []interface{}:
		y := YamlList{}
		for _, v := range val {
			y = append(y, toNode(v))
		}
		return y
	case map[string]interface{}:
		y := YamlMap{}
		for k, v := range val {
			y[k] = toNode(v)
		}
		return y
	case string:
		return YamlScalar(val)
	}

	return nil
} // }}}

func fmtJson(s string) string { // {{{
	r := ""
	for len(s) > 0 { // {{{
		idx := strings.IndexAny(s, "[]{}:,")
		if idx < 0 {
			return r + strings.Trim(s, " ")
		}

		_s := strings.Trim(s[:idx], " ")

		l := len(_s)

		if l == 0 {
			r = r + string(s[idx])
			s = s[idx+1:]
			continue
		}

		if l > 1 {
			if _s[0] == '\'' && _s[l-1] == '\'' {
				_s = strings.ReplaceAll(_s[1:l-1], `\'`, `'`)
				_s = `"` + strings.ReplaceAll(_s, `"`, `\"`) + `"`
			} else if !(_s[0] == '"' && _s[l-1] == '"') {
				_s = `"` + strings.ReplaceAll(_s, `"`, `\"`) + `"`
			}
		} else if l == 1 {
			_s = `"` + strings.ReplaceAll(_s, `"`, `\"`) + `"`
		}

		r = r + _s + string(s[idx])
		s = s[idx+1:]
	} // }}}

	return r
} // }}}
