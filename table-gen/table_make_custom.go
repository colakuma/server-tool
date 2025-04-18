package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

// 0
type IKeySlice interface {
	keySlice(int) bool
}

type IKeySliceName interface {
	keySliceName() []string
}

type IKeySliceSort interface {
	keySliceSort(any, int)
}

// 1
type IValueSlice interface {
	valueSlice(int) bool
}

type IValueSliceName interface {
	valueSliceName() []string
}

type IValueSliceSort interface {
	valueSliceSort(any, int)
}

// 2
type IFilterMap interface {
	filterMap(int) bool
}

type IFilterMapName interface {
	filterMapName() []string
}

type IAfterLoad interface {
	afterLoad(any)
}

type TabelCustomStrings struct {
	typePattern string
	varPattern  string
	varName     string
	typeName    string
	implPattern string
}

const customMax = 2

var (
	sliceIFunc     = []reflect.Type{reflect.TypeOf((*IKeySlice)(nil)).Elem(), reflect.TypeOf((*IValueSlice)(nil)).Elem(), reflect.TypeOf((*IFilterMap)(nil)).Elem()}
	sliceIFuncName = []reflect.Type{reflect.TypeOf((*IKeySliceName)(nil)).Elem(), reflect.TypeOf((*IValueSliceName)(nil)).Elem(), reflect.TypeOf((*IFilterMapName)(nil)).Elem()}
	sliceIFuncSort = []reflect.Type{reflect.TypeOf((*IKeySliceSort)(nil)).Elem(), reflect.TypeOf((*IValueSliceSort)(nil)).Elem(), reflect.TypeOf((*IFilterMapName)(nil)).Elem()}
	sliceFuncName  = []string{"keySliceName", "valueSliceName", ""}

	tabelCustomPattern = []TabelCustomStrings{
		{
			typePattern: "%sKeySlice\t%s\n\t",
			varPattern:  "sliceKey%s\tatomic.Pointer[%sKeySlice]\n\t",
			varName:     "sliceKey%s",
			typeName:    "%sKeySlice",
			implPattern: "keySlice",
		},
		{
			typePattern: "%sValueSlice\t%s\n\t",
			varPattern:  "sliceValue%s\tatomic.Pointer[%sValueSlice]\n\t",
			varName:     "sliceValue%s",
			typeName:    "%sValueSlice",
			implPattern: "valueSlice",
		},
		// {
		// 	typePattern: "%sFilterMap\t%s\n\t",
		// 	varPattern:  "mapFilter%s\tatomic.Value\n\t",
		// 	varName:     "mapFilter%s",
		// 	implPattern: "filterMap",
		// },
	}
)

func getCustomElemType(ts *TableStruct, i int) string {
	switch i {
	case 0:
		return "[]" + ts.keyType.String()
	case 1:
		return "[]*" + ts.typeName
		// case 2:
		// 	return fmt.Sprintf("map[%s]*%s", ts.keyType.String(), ts.typeName)
	}
	return ts.typeName
}

func MakeImpl(ts *TableStruct, varTmp, varName, name string, i, j int, hasSort bool, _make, _op, _append, _sort *[]string) {
	*_make = append(*_make, fmt.Sprintf(afterMake, varTmp, fmt.Sprintf(tabelCustomPattern[i].typeName, name))) // getCustomElemType(ts, i)))
	switch i {
	case 0:
		*_op = append(*_op, fmt.Sprintf(afterOp, tabelCustomPattern[i].implPattern, j, varTmp, varTmp, "k"))
	case 1:
		*_op = append(*_op, fmt.Sprintf(afterOp, tabelCustomPattern[i].implPattern, j, varTmp, varTmp, "v"))
		// case 2:
		// 	*_op = append(*_op, fmt.Sprintf(afterOpMap, tabelCustomPattern[i].implPattern, j, varTmp))
	}

	*_append = append(*_append, fmt.Sprintf(afterAppend, varName, varTmp))
	if hasSort {
		*_sort = append(*_sort, fmt.Sprintf(afterSort, ts.typeName, tabelCustomPattern[i].implPattern, varTmp, j))
	}
}

func makeCustomGet(i int, name, varName, typeName string, getMap map[string]string) {
	if i < 2 {
		getMap["getAllFunc"] += fmt.Sprintf(getCustomFunc, name, typeName, varName)
	}
}

// 新增函数：从源码解析结构体定义
func parseStructFromSource(structName string) (*ast.StructType, error) {
	// 1. 读取所有c_*.go文件
	files, err := filepath.Glob("./c_*.go")
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %v", err)
	}

	// 2. 解析每个文件查找目标结构体
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			log.Printf("warning: failed to parse file %s: %v", file, err)
			continue
		}

		// 3. 遍历文件中的声明
		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != structName {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// 确保结构体有字段
				if structType.Fields == nil || len(structType.Fields.List) == 0 {
					return nil, fmt.Errorf("struct %s has no fields", structName)
				}

				return structType, nil
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found in any c_*.go files", structName)
}

// 修改makeCustomOne函数
func makeCustomOne(ts *TableStruct, output map[string]string, getMap map[string]string) {
	// 替换反射方式为源码解析

	methodNames, _ := getStructMethods(ts.typeName)

	// 判断是否实现接口改为检查方法是否存在
	hasKeySlice := false
	if len(methodNames) > 0 && slices.Contains(methodNames, "keySlice") {
		hasKeySlice = true
	}
	hasValueSlice := false
	if len(methodNames) > 0 && slices.Contains(methodNames, "valueSlice") {
		hasValueSlice = true
	}
	hasAfterLoad := false
	if len(methodNames) > 0 && slices.Contains(methodNames, "afterLoad") {
		hasAfterLoad = true
	}

	_make := make([]string, 0)
	_op := make([]string, 0)
	_append := make([]string, 0)
	_sort := make([]string, 0)

	varIndex := 1
	hasK := false
	for i := 0; i < customMax; i++ {
		// 检查是否实现接口方法
		if (i == 0 && !hasKeySlice) || (i == 1 && !hasValueSlice) {
			continue
		}

		if i == 0 {
			hasK = true
		}

		// 获取自定义名称
		names := getCustomNamesFromSource(ts.typeName, i)
		isDefaultName := true
		if len(names) == 0 {
			isDefaultName = false
			names = append(names, ts.typeName)
		}

		bHasSort := false
		if len(methodNames) > 0 && slices.Contains(methodNames, fmt.Sprintf("%sSort", tabelCustomPattern[i].implPattern)) {
			bHasSort = true
		}

		for j, v := range names {
			output["typePattern"] += fmt.Sprintf(tabelCustomPattern[i].typePattern, v, getCustomElemType(ts, i))
			output["varPattern"] += fmt.Sprintf(tabelCustomPattern[i].varPattern, v, v)
			varTmp := fmt.Sprintf("var%d", varIndex)
			varName := fmt.Sprintf(tabelCustomPattern[i].varName, v)
			varIndex++
			MakeImpl(ts, varTmp, varName, v, i, j, bHasSort, &_make, &_op, &_append, &_sort)
			methodName := v
			if !isDefaultName {
				methodName = fmt.Sprintf(tabelCustomPattern[i].typeName, v)
			}
			makeCustomGet(i, methodName, varName, fmt.Sprintf(tabelCustomPattern[i].typeName, v), getMap)
		}
	}

	callStructAfterLoad := ""
	if hasAfterLoad {
		callStructAfterLoad = fmt.Sprintf(structAfterLoad, ts.typeName)
	}

	if len(_make) > 0 {
		strK := "_"
		if hasK {
			strK = "k"
		}
		output["implPattern"] += fmt.Sprintf(afterFunc, ts.typeName, ts.typeName, callStructAfterLoad, strings.Join(_make, "\n"), strK, strings.Join(_op, "\n"), strings.Join(_sort, "\n\t"), ts.varName, strings.Join(_append, "\n"))
	} else {
		output["implPattern"] += fmt.Sprintf(afterFunc2, ts.typeName, ts.typeName, callStructAfterLoad, ts.varName)
	}
}

// 从源码解析获取自定义名称
func getCustomNamesFromSource(structName string, i int) []string {
	// 1. 获取接口方法名
	var methodName string
	switch i {
	case 0:
		methodName = "keySliceName"
	case 1:
		methodName = "valueSliceName"
	default:
		return make([]string, 0)
	}

	// 2. 查找方法实现
	files, err := filepath.Glob("./c_*.go")
	if err != nil {
		return make([]string, 0)
	}

	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		// 3. 查找方法声明
		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || funcDecl.Name.Name != methodName {
				continue
			}

			// 4. 检查接收者类型是否匹配
			for _, recv := range funcDecl.Recv.List {
				starExpr, ok := recv.Type.(*ast.StarExpr)
				if !ok {
					continue
				}

				ident, ok := starExpr.X.(*ast.Ident)
				if !ok || ident.Name != structName {
					continue
				}

				// 5. 解析方法体获取返回的名称列表
				if stmt, ok := funcDecl.Body.List[0].(*ast.ReturnStmt); ok {
					if compLit, ok := stmt.Results[0].(*ast.CompositeLit); ok {
						var names []string
						for _, elt := range compLit.Elts {
							if bl, ok := elt.(*ast.BasicLit); ok {
								names = append(names, strings.Trim(bl.Value, `"`))
							}
						}
						return names
					}
				}
			}
		}
	}
	return make([]string, 0)
}

func makeCustom(lst []*TableStruct, getMap map[string]string) {
	output := make(map[string]string)
	for _, v := range lst {
		makeCustomOne(v, output, getMap)
	}

	WriteCustomGo(output, "./table_after_load.go")
}

func WriteCustomGo(output map[string]string, filePath string) {
	context := fmt.Sprintf(customFile, output["typePattern"], output["varPattern"], output["implPattern"])

	os.WriteFile(filePath, StringBytes(context), 0o600|0o064)
}

// 获取结构体的所有方法列表
func getStructMethods(structName string) ([]string, error) {
	// 存储找到的方法名
	methods := make([]string, 0)

	// 1. 读取所有c_*.go文件
	files, err := filepath.Glob("./c_*.go")
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %v", err)
	}

	// 2. 解析每个文件查找目标结构体的方法
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			log.Printf("warning: failed to parse file %s: %v", file, err)
			continue
		}

		// 3. 遍历文件中的声明，查找方法
		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil {
				continue // 不是方法或没有接收者
			}

			// 4. 检查接收者类型是否匹配
			for _, recv := range funcDecl.Recv.List {
				// 处理指针接收者 (*StructName)
				starExpr, isStarExpr := recv.Type.(*ast.StarExpr)
				if isStarExpr {
					if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == structName {
						methods = append(methods, funcDecl.Name.Name)
						break
					}
				} else if ident, ok := recv.Type.(*ast.Ident); ok && ident.Name == structName {
					// 处理值接收者 (StructName)
					methods = append(methods, funcDecl.Name.Name)
					break
				}
			}
		}
	}

	return methods, nil
}
