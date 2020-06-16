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
	PackagePath     string
	TypeName        string
	Methods         []Method
	PackageID       string
	PackageName     string
	TypeIsInterface bool
}

// ParseType ...
func (ms *MethodSet) ParseType(context *ParseContext, packagePath string, typeName string) error {
	ms.PackagePath = packagePath
	ms.TypeName = typeName
	var typeInfo typeInfo

	if _, err := ms.doParseType(context, "", packagePath, typeName, 0, &typeInfo); err != nil {
		return err
	}

	ms.PackageID = typeInfo.PackageID
	ms.PackageName = typeInfo.PackageName
	ms.TypeIsInterface = typeInfo.IsInterface
	return nil
}

type typeInfo struct {
	PackageID   string
	PackageName string
	IsInterface bool
}

func (ms *MethodSet) doParseType(
	context *ParseContext,
	currentPackageID string,
	packagePath string,
	typeName string,
	depth int,
	typeInfo *typeInfo,
) (_ bool, returnedErr error) {
	package1, err := context.importPackage(currentPackageID, packagePath)

	if err != nil {
		return false, err
	}

	defer func() {
		if returnedErr == nil {
			typeInfo.PackageID = package1.ID
			typeInfo.PackageName = package1.Name
		}
	}()

	object := package1.Scope.Lookup(typeName)

	if object == nil {
		return false, fmt.Errorf("methodset: type not found; packagePath=%q typeName=%q", packagePath, typeName)
	}

	if object.Kind != ast.Typ {
		return false, fmt.Errorf("methodset: non-type; packagePath=%q objectName=%q objectKind=%q",
			packagePath, object.Name, object.Kind)
	}

	typeSpec := object.Decl.(*ast.TypeSpec)

	switch type1 := typeSpec.Type.(type) {
	case *ast.Ident:
		if type1.Obj != nil && type1.Obj.Kind == ast.Typ {
			if packageBasicInfo, ok := context.getPackageBasicInfo(type1.Obj); ok {
				if typeSpec.Assign != token.NoPos {
					// type Foo = Bar
					return ms.doParseType(
						context,
						package1.ID,
						packageBasicInfo.Path,
						type1.Name,
						depth,
						typeInfo,
					)
				}
				// type Foo Bar

				if ok, err := ms.doParseType(
					context,
					package1.ID,
					packageBasicInfo.Path,
					type1.Name,
					depth+1,
					typeInfo,
				); err != nil || ok {
					return ok, err
				}
			} else {
				if typeSpec.Assign != token.NoPos {
					// type Foo = Bar
					return ms.parseBuiltinType(type1.Name, depth, typeInfo), nil
				}
				// type Foo Bar

				if ok := ms.parseBuiltinType(type1.Name, depth+1, typeInfo); ok {
					return true, nil
				}
			}
		} else {
			panic("unreachable")
		}
	case *ast.SelectorExpr:
		if x, ok := type1.X.(*ast.Ident); ok && x.Obj != nil && x.Obj.Kind == ast.Pkg {
			if packageBasicInfo, ok := context.getPackageBasicInfo(x.Obj); ok {
				if typeSpec.Assign != token.NoPos {
					// type Foo = Bar.Baz
					return ms.doParseType(
						context,
						package1.ID,
						packageBasicInfo.Path,
						type1.Sel.Name,
						depth,
						typeInfo,
					)
				}
				// type Foo Bar.Baz

				if ok, err := ms.doParseType(
					context,
					package1.ID,
					packageBasicInfo.Path,
					type1.Sel.Name,
					depth+1,
					typeInfo,
				); err != nil || ok {
					return ok, err
				}
			} else {
				panic("unreachable")
			}
		} else {
			panic("unreachable")
		}
	case *ast.InterfaceType:
		// type Foo interface { ... }
		typeInfo.IsInterface = true
		err = ms.parseInterfaceType(context, package1.ID, type1)
		return err == nil, err
	case *ast.StructType:
		// type Foo struct { ... }
		if err := ms.parseEmbeddedFields(context, package1.ID, type1.Fields); err != nil {
			return false, err
		}
	}

	if depth >= 1 {
		return false, nil
	}

	err = ms.parseFuncDecls(context, package1, object)
	return err == nil, err
}

func (ms *MethodSet) parseInterfaceType(context *ParseContext, currentPackageID string, interfaceType *ast.InterfaceType) error {
	for _, method1 := range interfaceType.Methods.List {
		switch methodType := method1.Type.(type) {
		case *ast.Ident:
			if methodType.Obj != nil && methodType.Obj.Kind == ast.Typ {
				if packageBasicInfo, ok := context.getPackageBasicInfo(methodType.Obj); ok {
					if _, err := ms.doParseType(
						context,
						currentPackageID,
						packageBasicInfo.Path,
						methodType.Name,
						0,
						&typeInfo{},
					); err != nil {
						return err
					}
				} else {
					ms.parseBuiltinType(methodType.Name, 0, &typeInfo{})
				}
			} else {
				panic("unreachable")
			}
		case *ast.SelectorExpr:
			if x, ok := methodType.X.(*ast.Ident); ok && x.Obj != nil && x.Obj.Kind == ast.Pkg {
				if packageBasicInfo, ok := context.getPackageBasicInfo(x.Obj); ok {
					if _, err := ms.doParseType(
						context,
						currentPackageID,
						packageBasicInfo.Path,
						methodType.Sel.Name,
						0,
						&typeInfo{},
					); err != nil {
						return err
					}
				} else {
					panic("unreachable")
				}
			} else {
				panic("unreachable")
			}
		case *ast.FuncType:
			if nameIsUnexported(method1.Names[0].Name) {
				continue
			}

			var method2 Method

			if err := method2.parseFuncType(context, method1.Names[0].Name, methodType); err != nil {
				return err
			}

			ms.addMethod(method2)
		default:
			panic("unreachable")
		}
	}

	return nil
}

func (ms *MethodSet) parseEmbeddedFields(context *ParseContext, currentPackageID string, fieldList *ast.FieldList) error {
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
			if fieldType.Obj != nil && fieldType.Obj.Kind == ast.Typ {
				if packageBasicInfo, ok := context.getPackageBasicInfo(fieldType.Obj); ok {
					if _, err := ms.doParseType(
						context,
						currentPackageID,
						packageBasicInfo.Path,
						fieldType.Name,
						0,
						&typeInfo{},
					); err != nil {
						return err
					}
				} else {
					ms.parseBuiltinType(fieldType.Name, 0, &typeInfo{})
				}
			} else {
				panic("unreachable")
			}
		case *ast.SelectorExpr:
			if x, ok := fieldType.X.(*ast.Ident); ok && x.Obj != nil && x.Obj.Kind == ast.Pkg {
				if packageBasicInfo, ok := context.getPackageBasicInfo(x.Obj); ok {
					if _, err := ms.doParseType(
						context,
						currentPackageID,
						packageBasicInfo.Path,
						fieldType.Sel.Name,
						0,
						&typeInfo{},
					); err != nil {
						return err
					}
				} else {
					panic("unreachable")
				}
			} else {
				panic("unreachable")
			}
		default:
			panic("unreachable")
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

			if nameIsUnexported(funcDecl.Name.Name) {
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
	restorers := [](func())(nil)

	visitor = visitorFunc(func(node ast.Node) ast.Visitor {
		switch type1 := node.(type) {
		case *ast.Ident:
			if type1.Obj != nil {
				if packageBasicInfo, ok := context.getPackageBasicInfo(type1.Obj); ok {
					backup := type1.Name
					restorers = append(restorers, func() { type1.Name = backup })
					type1.Name = "<PACKAGE>." + type1.Name
					t.PackageBasicInfos = append(t.PackageBasicInfos, packageBasicInfo)
				}
			}

			return nil
		case *ast.SelectorExpr:
			if x, ok := type1.X.(*ast.Ident); ok && x.Obj != nil && x.Obj.Kind == ast.Pkg {
				if packageBasicInfo, ok := context.getPackageBasicInfo(x.Obj); ok {
					backup := x.Name
					restorers = append(restorers, func() { x.Name = backup })
					x.Name = "<PACKAGE>"
					t.PackageBasicInfos = append(t.PackageBasicInfos, packageBasicInfo)
				}
			} else {
				panic("unreachable")
			}

			return nil
		}

		return visitor
	})

	ast.Walk(visitor, expr)
	buffer := bytes.NewBufferString("")
	printer.Fprint(buffer, context.FileSet(), expr)

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

func nameIsUnexported(name string) bool {
	return name[0] < 'A' || name[0] > 'Z'
}

var errorMethods = []Method{
	{
		Name: "Error",

		ResultTypes: []Type{{
			Format: "string",
		}},
	},
}
