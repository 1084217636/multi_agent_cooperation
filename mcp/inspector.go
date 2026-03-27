package mcp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CodeInfo 汇总项目中的静态结构信息。
type CodeInfo struct {
	Functions []FunctionInfo
	Structs   []StructInfo
	Packages  []string
	Imports   []ImportInfo
	Calls     []CallEdgeInfo
}

// FunctionInfo 描述函数定义。
type FunctionInfo struct {
	Name      string
	Package   string
	File      string
	Signature string
	Comment   string
	Line      int
}

// StructInfo 描述结构体定义。
type StructInfo struct {
	Name    string
	Package string
	File    string
	Fields  []FieldInfo
	Comment string
	Line    int
}

// FieldInfo 描述结构体字段。
type FieldInfo struct {
	Name    string
	Type    string
	Comment string
}

// ImportInfo 描述单个 import 声明。
type ImportInfo struct {
	Path    string
	Alias   string
	Package string
	File    string
}

// CallEdgeInfo 描述一条基础调用边。
type CallEdgeInfo struct {
	Caller  string
	Callee  string
	Package string
	File    string
	Line    int
}

// Inspector 负责扫描项目静态结构。
type Inspector struct {
	rootDir      string
	excludeNames map[string]struct{}
}

// NewInspector 创建代码检查器。
func NewInspector(rootDir string, excludeNames ...string) *Inspector {
	seen := make(map[string]struct{}, len(excludeNames))
	for _, name := range excludeNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	return &Inspector{rootDir: rootDir, excludeNames: seen}
}

// ScanProject 扫描项目代码。
func (i *Inspector) ScanProject() (*CodeInfo, error) {
	var codeInfo CodeInfo
	packages := make(map[string]bool)
	importSeen := make(map[string]bool)
	callSeen := make(map[string]bool)

	err := filepath.Walk(i.rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if _, ok := i.excludeNames[name]; ok {
				return filepath.SkipDir
			}
		}

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
			for _, item := range fileInfo.Imports {
				key := item.Package + "|" + item.File + "|" + item.Path + "|" + item.Alias
				if importSeen[key] {
					continue
				}
				importSeen[key] = true
				codeInfo.Imports = append(codeInfo.Imports, item)
			}
			for _, item := range fileInfo.Calls {
				key := item.Caller + "|" + item.Callee + "|" + item.File + fmt.Sprintf("|%d", item.Line)
				if callSeen[key] {
					continue
				}
				callSeen[key] = true
				codeInfo.Calls = append(codeInfo.Calls, item)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}

	for pkg := range packages {
		codeInfo.Packages = append(codeInfo.Packages, pkg)
	}
	sort.Strings(codeInfo.Packages)

	return &codeInfo, nil
}

func (i *Inspector) analyzeFile(filePath string) (*CodeInfo, string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	var codeInfo CodeInfo
	pkgName := node.Name.Name
	codeInfo.Imports = i.extractImports(node, filePath, pkgName)

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			funcInfo := i.extractFunctionInfo(d, fset, filePath, pkgName)
			codeInfo.Functions = append(codeInfo.Functions, funcInfo)
			codeInfo.Calls = append(codeInfo.Calls, i.extractCallEdges(d, fset, filePath, pkgName)...)
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

func (i *Inspector) extractImports(node *ast.File, filePath, pkgName string) []ImportInfo {
	var imports []ImportInfo
	for _, spec := range node.Imports {
		importInfo := ImportInfo{
			Path:    strings.Trim(spec.Path.Value, `"`),
			Package: pkgName,
			File:    filepath.Base(filePath),
		}
		if spec.Name != nil {
			importInfo.Alias = spec.Name.Name
		}
		imports = append(imports, importInfo)
	}
	return imports
}

func (i *Inspector) extractFunctionInfo(funcDecl *ast.FuncDecl, fset *token.FileSet, filePath, pkgName string) FunctionInfo {
	var comment string
	if funcDecl.Doc != nil {
		comment = funcDecl.Doc.Text()
	}

	return FunctionInfo{
		Name:      funcDecl.Name.Name,
		Package:   pkgName,
		File:      filepath.Base(filePath),
		Signature: i.formatFunctionSignature(funcDecl),
		Comment:   comment,
		Line:      fset.Position(funcDecl.Pos()).Line,
	}
}

func (i *Inspector) extractCallEdges(funcDecl *ast.FuncDecl, fset *token.FileSet, filePath, pkgName string) []CallEdgeInfo {
	if funcDecl.Body == nil {
		return nil
	}

	caller := i.functionDisplayName(funcDecl, pkgName)
	var edges []CallEdgeInfo
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		callee := i.formatCallTarget(call.Fun)
		if callee == "" {
			return true
		}

		edges = append(edges, CallEdgeInfo{
			Caller:  caller,
			Callee:  callee,
			Package: pkgName,
			File:    filepath.Base(filePath),
			Line:    fset.Position(call.Pos()).Line,
		})
		return true
	})
	return edges
}

func (i *Inspector) functionDisplayName(funcDecl *ast.FuncDecl, pkgName string) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return pkgName + "." + funcDecl.Name.Name
	}
	recv := strings.TrimPrefix(i.formatType(funcDecl.Recv.List[0].Type), "*")
	return pkgName + "." + recv + "." + funcDecl.Name.Name
}

func (i *Inspector) formatCallTarget(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return i.formatType(t.X) + "." + t.Sel.Name
	case *ast.IndexExpr:
		return i.formatCallTarget(t.X)
	case *ast.IndexListExpr:
		return i.formatCallTarget(t.X)
	default:
		return ""
	}
}

func (i *Inspector) formatFunctionSignature(funcDecl *ast.FuncDecl) string {
	var parts []string

	if funcDecl.Recv != nil {
		parts = append(parts, i.formatFieldList(funcDecl.Recv)+" ")
	}

	parts = append(parts, funcDecl.Name.Name)
	parts = append(parts, "(")
	if funcDecl.Type.Params != nil {
		parts = append(parts, i.formatFieldList(funcDecl.Type.Params))
	}
	parts = append(parts, ")")

	if funcDecl.Type.Results != nil {
		if len(funcDecl.Type.Results.List) == 1 && funcDecl.Type.Results.List[0].Names == nil {
			parts = append(parts, " ")
			parts = append(parts, i.formatType(funcDecl.Type.Results.List[0].Type))
		} else {
			parts = append(parts, " (")
			parts = append(parts, i.formatFieldList(funcDecl.Type.Results))
			parts = append(parts, ")")
		}
	}

	return strings.Join(parts, "")
}

func (i *Inspector) extractStructInfo(ts *ast.TypeSpec, structType *ast.StructType, fset *token.FileSet, filePath, pkgName string) StructInfo {
	var comment string
	if ts.Doc != nil {
		comment = ts.Doc.Text()
	}

	var fields []FieldInfo
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fields = append(fields, i.extractFieldInfo(field)...)
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

func (i *Inspector) extractFieldInfo(field *ast.Field) []FieldInfo {
	var fieldInfos []FieldInfo

	if field.Names == nil {
		fieldInfo := FieldInfo{Name: i.formatType(field.Type)}
		if field.Doc != nil {
			fieldInfo.Comment = field.Doc.Text()
		}
		fieldInfos = append(fieldInfos, fieldInfo)
		return fieldInfos
	}

	for _, name := range field.Names {
		fieldInfo := FieldInfo{
			Name: name.Name,
			Type: i.formatType(field.Type),
		}
		if field.Doc != nil {
			fieldInfo.Comment = field.Doc.Text()
		}
		fieldInfos = append(fieldInfos, fieldInfo)
	}

	return fieldInfos
}

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
			parts = append(parts, i.formatType(field.Type))
			continue
		}

		var names []string
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
		parts = append(parts, strings.Join(names, ", "))
		parts = append(parts, " ")
		parts = append(parts, i.formatType(field.Type))
	}

	return strings.Join(parts, "")
}

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

// GenerateSystemPrompt 生成静态结构提示。
func (i *Inspector) GenerateSystemPrompt() (string, error) {
	codeInfo, err := i.ScanProject()
	if err != nil {
		return "", err
	}

	var prompt strings.Builder
	prompt.WriteString("以下是项目中已有的代码结构信息，请在设计和实现时参考这些信息，避免创建重复或冲突的函数和结构体：\n\n")

	if len(codeInfo.Packages) > 0 {
		prompt.WriteString("## 包列表\n")
		for _, pkg := range codeInfo.Packages {
			prompt.WriteString(fmt.Sprintf("- %s\n", pkg))
		}
		prompt.WriteString("\n")
	}

	if len(codeInfo.Imports) > 0 {
		prompt.WriteString("## 导入样例\n")
		for _, item := range topImports(codeInfo.Imports, 10) {
			alias := ""
			if item.Alias != "" {
				alias = item.Alias + " "
			}
			prompt.WriteString(fmt.Sprintf("- %s%s (包: %s, 文件: %s)\n", alias, item.Path, item.Package, item.File))
		}
		prompt.WriteString("\n")
	}

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

	if len(codeInfo.Calls) > 0 {
		prompt.WriteString("## 调用关系样例\n")
		for _, edge := range topCalls(codeInfo.Calls, 12) {
			prompt.WriteString(fmt.Sprintf("- %s -> %s (%s:%d)\n", edge.Caller, edge.Callee, edge.File, edge.Line))
		}
		prompt.WriteString("\n")
	}

	return prompt.String(), nil
}

// GetFunctionByName 根据名称查找函数。
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

// GetStructByName 根据名称查找结构体。
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

func topImports(items []ImportInfo, limit int) []ImportInfo {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path == items[j].Path {
			return items[i].File < items[j].File
		}
		return items[i].Path < items[j].Path
	})
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func topCalls(items []CallEdgeInfo, limit int) []CallEdgeInfo {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Caller == items[j].Caller {
			if items[i].Callee == items[j].Callee {
				return items[i].Line < items[j].Line
			}
			return items[i].Callee < items[j].Callee
		}
		return items[i].Caller < items[j].Caller
	})
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}
