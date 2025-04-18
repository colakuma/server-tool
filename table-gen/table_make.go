package main

import (
	"fmt"
	"go/ast"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unsafe"
)

const (
	defaultKeyName = "id"
)

type TableStructRuntime struct {
	keyField  []*reflect.StructField
	keyType   reflect.Kind
	hasGetKey bool
	varName   string
}

func (p *TableStructRuntime) GenKeyParams() (string, string, string, string, bool) {
	if len(p.keyField) == 1 {
		return "key " + p.keyType.String(), "", "", "", p.keyType == reflect.Int
	}
	allInt := true
	params := make([]string, 0, len(p.keyField))
	placeHold := make([]string, 0, len(p.keyField))
	callParam := make([]string, 0, len(p.keyField))
	getKeyParam := make([]string, 0, len(p.keyField))
	for _, v := range p.keyField {
		params = append(params, GenKeyParam(v))
		callParam = append(callParam, firstCharLower(v.Name))
		getKeyParam = append(getKeyParam, "p."+v.Name)
		switch v.Type.Kind() {
		case reflect.Int:
			placeHold = append(placeHold, "%d")
		case reflect.String:
			placeHold = append(placeHold, "%s")
			allInt = false
		default:
			log.Fatalf("do not support key type, edit func (p *TableStructRuntime) GenKeyParams() code plz!")
		}

	}
	return strings.Join(params, ","), strings.Join(placeHold, "_"), strings.Join(callParam, ","), strings.Join(getKeyParam, ","), allInt
}

func firstCharLower(p string) string {
	tmp := strings.ToLower(p[:1]) + p[1:]
	if tmp == "type" {
		return "typ"
	}
	return tmp
}

func GenKeyParam(p *reflect.StructField) string {
	return firstCharLower(p.Name) + " " + p.Type.String()
}

type TableStruct struct {
	typeName string
	csv      string
	excel    string
	depend   []string
	TableStructRuntime
}

var (
	tables   map[string]*TableStruct
	tagReg   *regexp.Regexp
	feildReg *regexp.Regexp
	fatal    bool
)

func init() {
	fatal = false
	tables = make(map[string]*TableStruct)
	tagReg = regexp.MustCompile(`(?sU)/\*(.+)\*/`)
	feildReg = regexp.MustCompile(`@(.+)[^\r\n]`)
}

func GetKeyType(kind reflect.Kind, allIntKey bool, keySize int) string {
	if allIntKey {
		switch keySize {
		case 1:
			return kind.String()
		case 2:
			return "tools.Key2"
		case 3:
			return "tools.Key3"
		default:
			return "string"
		}
	}
	return kind.String()
}

func makeTableStructStuff(ts *TableStruct, output map[string]string) {
	fillKey(ts)

	p, h, c, getKey, allIntKey := ts.GenKeyParams()
	key := ""
	if h != "" {
		if allIntKey {
			key = fmt.Sprintf(keyMake2, len(ts.keyField), c)
		} else {
			key = fmt.Sprintf(keyMake, h, c)
		}
	}

	output["mapType"] += fmt.Sprintf(mapType, ts.typeName, GetKeyType(ts.keyType, allIntKey, len(ts.keyField)), ts.typeName)
	output["mapVar"] += fmt.Sprintf(mapVar, ts.varName, ts.typeName)
	output["loadFunc"] += fmt.Sprintf(loadFunc, ts.typeName, ts.typeName, ts.excel, ts.typeName, ts.csv, ts.typeName, ts.excel, ts.csv, ts.typeName)
	output["getFunc"] += fmt.Sprintf(getFunc, ts.typeName, p, ts.typeName, key, ts.varName)
	output["getAllFunc"] += fmt.Sprintf(getAllFunc, ts.typeName, ts.typeName, ts.varName)
	output["callLoad"] += fmt.Sprintf(callLoad, ts.typeName)
	if ts.hasGetKey {
		if len(ts.keyField) > 1 {
			if allIntKey {
				output["getKey"] += fmt.Sprintf(getKeyNPattern, ts.typeName, len(ts.keyField), getKey)
			} else {
				output["getKey"] += fmt.Sprintf(getKeyStringPattern, ts.typeName, h, getKey)
			}
		} else {
			output["getKey"] += fmt.Sprintf(getKeyPattern, ts.typeName, ts.keyField[0].Name)
		}
	}
}

func makeTableVar(ts *TableStruct, output *[]string, output2 *[]string) {
	*output = append(*output, fmt.Sprintf("dummy%s *%s", ts.typeName, ts.typeName))
	*output2 = append(*output2, fmt.Sprintf("dummy%s = &%s{}", ts.typeName, ts.typeName))
}

func StringBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

func WriteTableGo(output map[string]string, filePath string) {
	loadAll := fmt.Sprintf(loadAllFunc, output["callLoad"])
	context := fmt.Sprintf(fileContext, *basePkg, output["mapType"], output["mapVar"], loadAll, output["loadFunc"], output["getKey"], output["getFunc"], output["getAllFunc"])

	os.WriteFile(filePath, StringBytes(context), 0o600|0o064)
}

func WriteVarGo(output []string, output2 []string, filePath string) {
	context := fmt.Sprintf(varFile, strings.Join(output, "\n\t"), strings.Join(output2, "\n\t"))

	os.WriteFile(filePath, StringBytes(context), 0o600|0o064)
}

func fillKey(ts *TableStruct) {
	realName := ts.typeName

	structType, err := parseStructFromSource(realName)
	if err != nil {
		log.Fatalf("struct type not exits3:%s! error: %v", realName, err)
	}

	hasId := false
	ts.keyField = make([]*reflect.StructField, 0)

	var defaultSf *reflect.StructField

	// 从结构体定义中解析字段
	for _, field := range structType.Fields.List {
		// 跳过匿名/嵌入字段
		if len(field.Names) == 0 {
			continue
		}
		fieldName := field.Names[0].Name
		fieldType := field.Type

		// 统一转成小写比较
		if strings.ToLower(fieldName) == defaultKeyName {
			hasId = true
			// 构建默认key字段

			defaultSf = &reflect.StructField{
				Name: fieldName,
				Type: getReflectTypeFromAst(fieldType),
				Tag:  reflect.StructTag(""),
			}
		}

		// 检查字段标签
		if field.Tag != nil {
			tag := field.Tag.Value
			// 更完善的标签解析逻辑
			if strings.HasPrefix(tag, "`gtable:") {
				// 去除反引号
				tag = strings.Trim(tag, "`")
				// 提取gtable标签内容
				tagStr := strings.TrimPrefix(tag, `gtable:`)
				tagStr = strings.Trim(tagStr, `"`)
				// 分割标签属性
				attrs := strings.Split(tagStr, ",")
				for _, attr := range attrs {
					attr = strings.TrimSpace(attr)
					if strings.Contains(attr, "key") {
						// 构建 StructField 对象
						sf := &reflect.StructField{
							Name: fieldName,
							Type: getReflectTypeFromAst(fieldType),
							Tag:  reflect.StructTag(tag),
						}
						ts.keyField = append(ts.keyField, sf)
						break
					}
				}
			}
		}
	}

	if len(ts.keyField) == 0 {
		if !hasId {
			log.Printf("1 struct %s has not specific key or default key [%s]!", realName, defaultKeyName)
			fatal = true
			return
		}
		ts.keyField = append(ts.keyField, defaultSf)
	} else {
		ts.hasGetKey = true
	}

	if len(ts.keyField) > 1 {
		ts.keyType = reflect.String
	} else {
		ts.keyType = ts.keyField[0].Type.Kind()
	}

	ts.varName = fmt.Sprintf(varName, ts.typeName)
}

func enumFile(dirPth string, prefix string) (files []string, err error) {
	files = make([]string, 0, 10)
	dir, err := os.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	pthSep := string(os.PathSeparator)
	prefix = strings.ToUpper(prefix)
	for _, fi := range dir {
		if fi.IsDir() {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(fi.Name()), prefix) {
			files = append(files, dirPth+pthSep+fi.Name())
		}
	}
	return files, nil
}

func readAll(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return io.ReadAll(f)
}

func walkFile(path string) {
	// fmt.Println(path)
	fc, err := readAll(path)
	if err != nil {
		return
	}

	lst := tagReg.FindAllString(string(fc), -1)
	for _, v := range lst {
		parseTags(v)
	}
}

func parseTag(tagStr string) []string {
	lst := strings.Split(tagStr, " ")
	if len(lst) != 2 {
		fmt.Printf("incorrect tag string:%s\n", tagStr)
	}
	return lst
}

func parseTags(tagStr string) {
	lst := feildReg.FindAllStringSubmatch(string(tagStr), -1)
	if len(lst) == 0 {
		return
	}
	typName := string(lst[0][0][1:])
	name := strings.ToLower(typName)
	t := &TableStruct{typeName: typName}
	lst = lst[1:]
	for _, v := range lst {
		tags := parseTag(v[0])
		switch tags[0][1:] {
		case "csv":
			t.csv = tags[1] // strings.ToLower(tags[1])
		case "excel":
			t.excel = tags[1]
		case "depend":
			t.depend = append(t.depend, strings.Split(tags[1], "|")...)
		}
	}
	tables[name] = t
}

func makeTableGo() {
	output := make(map[string]string)
	lst := make([]*TableStruct, 0, len(tables))
	for _, v := range tables {
		lst = append(lst, v)
	}

	log.Printf("----------%d", len(lst))

	sort.Slice(lst, func(i, j int) bool {
		return lst[i].typeName < lst[j].typeName
	})

	for _, v := range lst {
		makeTableStructStuff(v, output)
	}
	if fatal {
		panic("fatal error!")
	}

	makeCustom(lst, output)
	WriteTableGo(output, "./table.go")
}

func makeVar() {
	output := make([]string, 0, len(tables))
	output2 := make([]string, 0, len(tables))
	for _, v := range tables {
		makeTableVar(v, &output, &output2)
	}

	WriteVarGo(output, output2, "./var.go")
}

// 新增辅助函数：从AST类型节点获取reflect.Type
func getReflectTypeFromAst(expr ast.Expr) reflect.Type {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "int":
			return reflect.TypeOf(0)
		case "string":
			return reflect.TypeOf("")
		// 添加更多类型支持...
		default:
			return reflect.TypeOf((*interface{})(nil)).Elem()
		}
	case *ast.StarExpr:
		return reflect.PtrTo(getReflectTypeFromAst(t.X))
	// 添加更多类型支持...
	default:
		return reflect.TypeOf((*interface{})(nil)).Elem()
	}
}
