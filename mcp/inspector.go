package mcp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// CodeInfo 代码信息
type CodeInfo struct {
	Functions []FunctionInfo
	Structs   []StructInfo
	Packages  []string
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name        string
	Package     string
	File        string
	Signature   string
	Comment     string
	Line        int
}

// StructInfo 结构体信息
type StructInfo struct {
	Name      string
	Package   string
	File      string
	Fields    []FieldInfo
	Comment   string
	Line      int
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name    string
	Type    string
	Comment string
}

// Inspector 代码检查器
type Inspector struct {
	rootDir string
}

// NewInspector 创建代码检查器
func NewInspector(rootDir string) *Inspector {
	return &Inspector{rootDir: rootDir}
}

// ScanProject 扫描项目代码
func (i *Inspector) ScanProject() (*CodeInfo, error) {
	var codeInfo CodeInfo
	packages := make(map[string]bool)

	err := filepath.Walk(i.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏目录
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// 只处理.go文件
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			fileInfo, pkgName, err := i.analyzeFile(path)
			if err != nil {
				return err
			}

			if pkgName != "" {
				packages[pkgName] = true
			}

			codeInfo.Functions = append(codeInfo.Functions, fileInfo.Functions...)
			codeInfo.Structs = append(codeInfo.Structs, fileInfo.Structs...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}

	// 转换包名映射为切片
	for pkg := range packages {
		codeInfo.Packages = append(codeInfo.Packages, pkg)
	}

	return &codeInfo, nil
}

// analyzeFile 分析单个文件
func (i *Inspector) analyzeFile(filePath string) (*CodeInfo, string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	var codeInfo CodeInfo
	pkgName := node.Name.Name

	// 遍历文件中的所有声明
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			funcInfo := i.extractFunctionInfo(d, fset, filePath, pkgName)
			codeInfo.Functions = append(codeInfo.Functions, funcInfo)
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := ts.Type.(*ast.StructType); ok {
							structInfo := i.extractStructInfo(ts, structType, fset, filePath, pkgName)
							codeInfo.Structs = append(codeInfo.Structs, structInfo)
						}
					}
				}
			}
		}
	}

	return &codeInfo, pkgName, nil
}

// extractFunctionInfo 提取函数信息
func (i *Inspector) extractFunctionInfo(funcDecl *ast.FuncDecl, fset *token.FileSet, filePath, pkgName string) FunctionInfo {
	var comment string
	if funcDecl.Doc != nil {
		comment = funcDecl.Doc.Text()
	}

	signature := i.formatFunctionSignature(funcDecl)

	return FunctionInfo{
		Name:      funcDecl.Name.Name,
		Package:   pkgName,
		File:      filepath.Base(filePath),
		Signature: signature,
		Comment:   comment,
		Line:      fset.Position(funcDecl.Pos()).Line,
	}
}

// formatFunctionSignature 格式化函数签名
func (i *Inspector) formatFunctionSignature(funcDecl *ast.FuncDecl) string {
	var parts []string

	// 接收者
	if funcDecl.Recv != nil {
		recvStr := i.formatFieldList(funcDecl.Recv)
		parts = append(parts, recvStr+" ")
	}

	// 函数名
	parts = append(parts, funcDecl.Name.Name)

	// 参数
	parts = append(parts, "(")
	if funcDecl.Type.Params != nil {
		parts = append(parts, i.formatFieldList(funcDecl.Type.Params))
	}
	parts = append(parts, ")")

	// 返回值
	if funcDecl.Type.Results != nil {
		if len(funcDecl.Type.Results.List) == 1 && funcDecl.Type.Results.List[0].Names == nil {
			// 单个无名称返回值
			parts = append(parts, " ")
			parts = append(parts, i.formatType(funcDecl.Type.Results.List[0].Type))
		} else {
			// 多个返回值或有名称的返回值
			parts = append(parts, " (")
			parts = append(parts, i.formatFieldList(funcDecl.Type.Results))
			parts = append(parts, ")")
		}
	}

	return strings.Join(parts, "")
}

// extractStructInfo 提取结构体信息
func (i *Inspector) extractStructInfo(ts *ast.TypeSpec, structType *ast.StructType, fset *token.FileSet, filePath, pkgName string) StructInfo {
	var comment string
	if ts.Doc != nil {
		comment = ts.Doc.Text()
	}

	var fields []FieldInfo
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fieldInfo := i.extractFieldInfo(field)
			fields = append(fields, fieldInfo...)
		}
	}

	return StructInfo{
		Name:    ts.Name.Name,
		Package: pkgName,
		File:    filepath.Base(filePath),
		Fields:  fields,
		Comment: comment,
		Line:    fset.Position(ts.Pos()).Line,
	}
}

// extractFieldInfo 提取字段信息
func (i *Inspector) extractFieldInfo(field *ast.Field) []FieldInfo {
	var fieldInfos []FieldInfo

	if field.Names == nil {
		// 嵌入字段
		fieldInfo := FieldInfo{
			Name:    i.formatType(field.Type),
			Type:    "",
			Comment: "",
		}
		if field.Doc != nil {
			fieldInfo.Comment = field.Doc.Text()
		}
		fieldInfos = append(fieldInfos, fieldInfo)
	} else {
		// 普通字段
		for _, name := range field.Names {
			fieldInfo := FieldInfo{
				Name:    name.Name,
				Type:    i.formatType(field.Type),
				Comment: "",
			}
			if field.Doc != nil {
				fieldInfo.Comment = field.Doc.Text()
			}
			fieldInfos = append(fieldInfos, fieldInfo)
		}
	}

	return fieldInfos
}

// formatFieldList 格式化字段列表
func (i *Inspector) formatFieldList(fieldList *ast.FieldList) string {
	if fieldList == nil {
		return ""
	}

	var parts []string
	for j, field := range fieldList.List {
		if j > 0 {
			parts = append(parts, ", ")
		}

		if field.Names == nil {
			// 嵌入字段
			parts = append(parts, i.formatType(field.Type))
		} else {
			// 普通字段
			var names []string
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
			parts = append(parts, strings.Join(names, ", "))
			parts = append(parts, " ")
			parts = append(parts, i.formatType(field.Type))
		}
	}

	return strings.Join(parts, "")
}

// formatType 格式化类型
func (i *Inspector) formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + i.formatType(t.X)
	case *ast.ArrayType:
		return "[]" + i.formatType(t.Elt)
	case *ast.MapType:
		return "map[" + i.formatType(t.Key) + "]" + i.formatType(t.Value)
	case *ast.ChanType:
		dir := ""
		switch t.Dir {
		case ast.SEND:
			dir = "<-"
		case ast.RECV:
			dir = "<-"
		case ast.SEND | ast.RECV:
			dir = "<-"
		}
		if t.Dir == ast.SEND {
			return "chan" + dir + i.formatType(t.Value)
		}
		return dir + "chan " + i.formatType(t.Value)
	case *ast.FuncType:
		var parts []string
		parts = append(parts, "func(")
		if t.Params != nil {
			parts = append(parts, i.formatFieldList(t.Params))
		}
		parts = append(parts, ")")

		if t.Results != nil {
			if len(t.Results.List) == 1 && t.Results.List[0].Names == nil {
				parts = append(parts, " ")
				parts = append(parts, i.formatType(t.Results.List[0].Type))
			} else {
				parts = append(parts, " (")
				parts = append(parts, i.formatFieldList(t.Results))
				parts = append(parts, ")")
			}
		}
		return strings.Join(parts, "")
	case *ast.SelectorExpr:
		return i.formatType(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", t)
	}
}

// GenerateSystemPrompt 生成系统提示
func (i *Inspector) GenerateSystemPrompt() (string, error) {
	codeInfo, err := i.ScanProject()
	if err != nil {
		return "", err
	}

	var prompt strings.Builder
	prompt.WriteString("以下是项目中已有的代码结构信息，请在设计和实现时参考这些信息，避免创建重复或冲突的函数和结构体：\n\n")

	// 包信息
	if len(codeInfo.Packages) > 0 {
		prompt.WriteString("## 包列表\n")
		for _, pkg := range codeInfo.Packages {
			prompt.WriteString(fmt.Sprintf("- %s\n", pkg))
		}
		prompt.WriteString("\n")
	}

	// 结构体信息
	if len(codeInfo.Structs) > 0 {
		prompt.WriteString("## 结构体定义\n")
		for _, structInfo := range codeInfo.Structs {
			prompt.WriteString(fmt.Sprintf("### %s (文件: %s, 包: %s)\n", structInfo.Name, structInfo.File, structInfo.Package))
			if structInfo.Comment != "" {
				prompt.WriteString(fmt.Sprintf("注释: %s\n", structInfo.Comment))
			}
			if len(structInfo.Fields) > 0 {
				prompt.WriteString("字段:\n")
				for _, field := range structInfo.Fields {
					if field.Type != "" {
						prompt.WriteString(fmt.Sprintf("- %s %s", field.Name, field.Type))
					} else {
						prompt.WriteString(fmt.Sprintf("- %s", field.Name))
					}
					if field.Comment != "" {
						prompt.WriteString(fmt.Sprintf(" // %s", field.Comment))
					}
					prompt.WriteString("\n")
				}
			}
			prompt.WriteString("\n")
		}
	}

	// 函数信息
	if len(codeInfo.Functions) > 0 {
		prompt.WriteString("## 函数定义\n")
		for _, funcInfo := range codeInfo.Functions {
			prompt.WriteString(fmt.Sprintf("### %s (文件: %s, 包: %s, 行号: %d)\n", funcInfo.Name, funcInfo.File, funcInfo.Package, funcInfo.Line))
			prompt.WriteString(fmt.Sprintf("签名: %s\n", funcInfo.Signature))
			if funcInfo.Comment != "" {
				prompt.WriteString(fmt.Sprintf("注释: %s\n", funcInfo.Comment))
			}
			prompt.WriteString("\n")
		}
	}

	return prompt.String(), nil
}

// GetFunctionByName 根据名称查找函数
func (i *Inspector) GetFunctionByName(name string) (*FunctionInfo, error) {
	codeInfo, err := i.ScanProject()
	if err != nil {
		return nil, err
	}

	for _, funcInfo := range codeInfo.Functions {
		if funcInfo.Name == name {
			return &funcInfo, nil
		}
	}

	return nil, fmt.Errorf("function %s not found", name)
}

// GetStructByName 根据名称查找结构体
func (i *Inspector) GetStructByName(name string) (*StructInfo, error) {
	codeInfo, err := i.ScanProject()
	if err != nil {
		return nil, err
	}

	for _, structInfo := range codeInfo.Structs {
		if structInfo.Name == name {
			return &structInfo, nil
		}
	}

	return nil, fmt.Errorf("struct %s not found", name)
}