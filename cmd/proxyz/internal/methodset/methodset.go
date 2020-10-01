package methodset

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

// MethodSet ...
type MethodSet struct {
	PackagePattern  string
	TypeName        string
	Methods         []Method
	PackageName     string
	PackagePath     string
	ImportPath      string
	TypeIsInterface bool
}

// ParseType ...
func (ms *MethodSet) ParseType(context *ParseContext, packagePattern string, typeName string) error {
	ms.PackagePattern = packagePattern
	ms.TypeName = typeName
	var typeInfo typeInfo

	if _, err := ms.doParseType(context, "", packagePattern, typeName, 0, &typeInfo); err != nil {
		return err
	}

	ms.PackageName = typeInfo.PackageName
	ms.PackagePath = typeInfo.PackagePath
	ms.ImportPath = packagePathToImportPath(typeInfo.PackagePath)
	ms.TypeIsInterface = typeInfo.IsInterface
	return nil
}

type typeInfo struct {
	PackageName string
	PackagePath string
	ImportPath  string
	IsInterface bool
}

func (ms *MethodSet) doParseType(
	context *ParseContext,
	currentPackagePath string,
	importPathOrPackagePattern string,
	typeName string,
	depth int,
	typeInfo *typeInfo,
) (_ bool, returnedErr error) {
	package1, err := context.importPackage(currentPackagePath, importPathOrPackagePattern)

	if err != nil {
		return false, err
	}

	defer func() {
		if returnedErr == nil {
			typeInfo.PackageName = package1.Name
			typeInfo.PackagePath = package1.PkgPath
			typeInfo.ImportPath = package1.ImportPath
		}
	}()

	object := package1.Scope.Lookup(typeName)

	if object == nil {
		return false, fmt.Errorf("methodset: type not found; packagePath=%q typeName=%q",
			package1.PkgPath, typeName)
	}

	if object.Kind != ast.Typ {
		return false, fmt.Errorf("methodset: non-type; packagePath=%q objectName=%q objectKind=%q",
			package1.PkgPath, object.Name, object.Kind)
	}

	typeSpec := object.Decl.(*ast.TypeSpec)

	switch type1 := typeSpec.Type.(type) {
	case *ast.Ident:
		object := type1.Obj

		if object != nil && object.Kind == ast.Typ {
			if importPath, ok := context.getImportPath(object); ok {
				if typeSpec.Assign != token.NoPos {
					// type Foo = Bar
					return ms.doParseType(
						context,
						package1.PkgPath,
						importPath,
						object.Name,
						depth,
						typeInfo,
					)
				}
				// type Foo Bar

				if ok, err := ms.doParseType(
					context,
					package1.PkgPath,
					importPath,
					object.Name,
					depth+1,
					typeInfo,
				); err != nil || ok {
					return ok, err
				}
			} else {
				if object.Decl == nil {
					// builtin type
					if typeSpec.Assign != token.NoPos {
						// type Foo = Bar
						return ms.parseBuiltinType(object.Name, depth, typeInfo), nil
					}
					// type Foo Bar

					if ok := ms.parseBuiltinType(object.Name, depth+1, typeInfo); ok {
						return true, nil
					}
				}
			}
		}
	case *ast.SelectorExpr:
		object := type1.Sel.Obj

		if object != nil && object.Kind == ast.Typ {
			if importPath, ok := context.getImportPath(object); ok {
				if typeSpec.Assign != token.NoPos {
					// type Foo = Bar.Baz
					return ms.doParseType(
						context,
						package1.PkgPath,
						importPath,
						object.Name,
						depth,
						typeInfo,
					)
				}
				// type Foo Bar.Baz

				if ok, err := ms.doParseType(
					context,
					package1.PkgPath,
					importPath,
					object.Name,
					depth+1,
					typeInfo,
				); err != nil || ok {
					return ok, err
				}
			}
		}
	case *ast.InterfaceType:
		// type Foo interface { ... }
		typeInfo.IsInterface = true
		err = ms.parseInterfaceType(context, package1.PkgPath, type1)
		return err == nil, err
	case *ast.StructType:
		// type Foo struct { ... }
		if err := ms.parseEmbeddedFields(context, package1.PkgPath, type1.Fields); err != nil {
			return false, err
		}
	}

	if depth >= 1 {
		return false, nil
	}

	err = ms.parseFuncDecls(context, package1, object)
	return err == nil, err
}

func (ms *MethodSet) parseInterfaceType(context *ParseContext, currentPackagePath string, interfaceType *ast.InterfaceType) error {
	for _, method1 := range interfaceType.Methods.List {
		switch methodType := method1.Type.(type) {
		case *ast.Ident:
			object := methodType.Obj

			if object != nil && object.Kind == ast.Typ {
				if importPath, ok := context.getImportPath(object); ok {
					if _, err := ms.doParseType(
						context,
						currentPackagePath,
						importPath,
						object.Name,
						1,
						&typeInfo{},
					); err != nil {
						return err
					}
				} else {
					if object.Decl == nil {
						// builtin type
						ms.parseBuiltinType(object.Name, 1, &typeInfo{})
					}
				}
			}
		case *ast.SelectorExpr:
			object := methodType.Sel.Obj

			if object != nil && object.Kind == ast.Typ {
				if importPath, ok := context.getImportPath(object); ok {
					if _, err := ms.doParseType(
						context,
						currentPackagePath,
						importPath,
						object.Name,
						1,
						&typeInfo{},
					); err != nil {
						return err
					}
				}
			}
		case *ast.FuncType:
			if !nameIsExported(method1.Names[0].Name) {
				continue
			}

			var method2 Method

			if err := method2.parseFuncType(context, method1.Names[0].Name, methodType); err != nil {
				return err
			}

			ms.addMethod(method2)
		}
	}

	return nil
}

func (ms *MethodSet) parseEmbeddedFields(context *ParseContext, currentPackagePath string, fieldList *ast.FieldList) error {
	for _, field := range fieldList.List {
		if len(field.Names) >= 1 {
			continue
		}

		fieldType := field.Type

		if starExpr, ok := fieldType.(*ast.StarExpr); ok {
			fieldType = starExpr.X
		}

		switch fieldType := fieldType.(type) {
		case *ast.Ident:
			object := fieldType.Obj

			if object != nil && object.Kind == ast.Typ {
				if importPath, ok := context.getImportPath(object); ok {
					if _, err := ms.doParseType(
						context,
						currentPackagePath,
						importPath,
						object.Name,
						0,
						&typeInfo{},
					); err != nil {
						return err
					}
				} else {
					if object.Decl == nil {
						// builtin type
						ms.parseBuiltinType(fieldType.Name, 0, &typeInfo{})
					}
				}
			}
		case *ast.SelectorExpr:
			object := fieldType.Sel.Obj

			if object != nil && object.Kind == ast.Typ {
				if importPath, ok := context.getImportPath(object); ok {
					if _, err := ms.doParseType(
						context,
						currentPackagePath,
						importPath,
						object.Name,
						0,
						&typeInfo{},
					); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (ms *MethodSet) parseFuncDecls(context *ParseContext, package1 *package1, object *ast.Object) error {
	for _, file := range package1.Syntax {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)

			if !ok {
				continue
			}

			recv := funcDecl.Recv

			if recv == nil {
				continue
			}

			if len(recv.List) == 0 {
				continue
			}

			recvType := recv.List[0].Type

			if starExpr, ok := recvType.(*ast.StarExpr); ok {
				recvType = starExpr.X
			}

			if recvType, ok := recvType.(*ast.Ident); ok {
				if recvType.Obj != object {
					continue
				}
			} else {
				continue
			}

			if !nameIsExported(funcDecl.Name.Name) {
				continue
			}

			var method Method

			if err := method.parseFuncType(context, funcDecl.Name.Name, funcDecl.Type); err != nil {
				return err
			}

			ms.addMethod(method)
		}
	}

	return nil
}

func (ms *MethodSet) parseBuiltinType(typeName string, depth int, typeInfo *typeInfo) bool {
	switch typeName {
	case "error":
		typeInfo.IsInterface = true

		for i := range errorMethods {
			ms.addMethod(errorMethods[i])
		}

		return true
	}

	return depth == 0
}

func (ms *MethodSet) addMethod(method1 Method) {
	for i, method2 := range ms.Methods {
		if method2.Name == method1.Name {
			ms.Methods[i] = method1
			return
		}
	}

	ms.Methods = append(ms.Methods, method1)
}

// Method ...
type Method struct {
	Name        string
	ArgNames    []string
	ArgTypes    []Type
	ResultTypes []Type
	IsVariadic  bool
}

func (m *Method) parseFuncType(context *ParseContext, name string, funcType *ast.FuncType) error {
	m.Name = name

	for _, param := range funcType.Params.List {
		if len(param.Names) >= 1 {
			m.ArgNames = append(m.ArgNames, param.Names[0].Name)
		}

		paramType := param.Type

		if ellipsis, ok := paramType.(*ast.Ellipsis); ok {
			m.IsVariadic = true
			paramType = ellipsis.Elt
		}

		var type1 Type

		if err := type1.parseExpr(context, paramType); err != nil {
			return err
		}

		m.ArgTypes = append(m.ArgTypes, type1)

		for i := 1; i < len(param.Names); i++ {
			m.ArgNames = append(m.ArgNames, param.Names[i].Name)
			m.ArgTypes = append(m.ArgTypes, type1)
		}
	}

	if results := funcType.Results; results != nil {
		for _, result := range results.List {
			var type1 Type

			if err := type1.parseExpr(context, result.Type); err != nil {
				return err
			}

			m.ResultTypes = append(m.ResultTypes, type1)

			for i := 1; i < len(result.Names); i++ {
				m.ResultTypes = append(m.ResultTypes, type1)
			}
		}
	}

	return nil
}

// Type ...
type Type struct {
	Format            string
	PackageBasicInfos []PackageBasicInfo
}

func (t *Type) parseExpr(context *ParseContext, expr ast.Expr) error {
	var visitor ast.Visitor
	var restorers []func()

	visitor = visitorFunc(func(node ast.Node) ast.Visitor {
		switch node := node.(type) {
		case *ast.Ident:
			if node.Obj != nil {
				if packageBasicInfo, ok := context.getPackageBasicInfo(node.Obj); ok {
					backup := node.Name
					restorers = append(restorers, func() { node.Name = backup })
					node.Name = "<PACKAGE>." + node.Name
					t.PackageBasicInfos = append(t.PackageBasicInfos, packageBasicInfo)
				}
			}

			return nil
		case *ast.SelectorExpr:
			if x, ok := node.X.(*ast.Ident); ok && x.Obj != nil && x.Obj.Kind == ast.Pkg {
				if packageBasicInfo, ok := context.getPackageBasicInfo(node.Sel.Obj); ok {
					backup := x.Name
					restorers = append(restorers, func() { x.Name = backup })
					x.Name = "<PACKAGE>"
					t.PackageBasicInfos = append(t.PackageBasicInfos, packageBasicInfo)
				}
			}

			return nil
		}

		return visitor
	})

	ast.Walk(visitor, expr)
	var buffer bytes.Buffer
	printer.Fprint(&buffer, context.FileSet(), expr)

	for _, restorer := range restorers {
		restorer()
	}

	t.Format = strings.ReplaceAll(buffer.String(), "<PACKAGE>.", "%s")
	return nil
}

type visitorFunc func(ast.Node) ast.Visitor

func (vf visitorFunc) Visit(node ast.Node) ast.Visitor {
	return vf(node)
}

var errorMethods = []Method{
	{
		Name: "Error",

		ResultTypes: []Type{{
			Format: "string",
		}},
	},
}

func nameIsExported(name string) bool {
	return name[0] >= 'A' && name[0] <= 'Z'
}
