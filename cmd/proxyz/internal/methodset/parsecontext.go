package methodset

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"strconv"
	"strings"

	"errors"

	"golang.org/x/tools/go/packages"
)

// ParseContext ...
type ParseContext struct {
	fileSet             *token.FileSet
	packagePath2Package map[string]*package1
	object2Package      map[*ast.Object]*package1
}

// Init ...
func (pc *ParseContext) Init() *ParseContext {
	pc.fileSet = token.NewFileSet()
	pc.packagePath2Package = map[string]*package1{}
	pc.object2Package = map[*ast.Object]*package1{}
	return pc
}

// FileSet ...
func (pc *ParseContext) FileSet() *token.FileSet {
	return pc.fileSet
}

func (pc *ParseContext) importPackage(currentPackagePath string, importPathOrPackagePattern string) (*package1, error) {
	var rawPackage *packages.Package
	var importPath string

	if currentPackagePath == "" {
		packagePattern := importPathOrPackagePattern

		rawPackages, err := packages.Load(&packages.Config{
			Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedSyntax,
			Fset: pc.fileSet,
		}, packagePattern)

		if err != nil {
			return nil, fmt.Errorf("methodset: package load failed; packagePattern=%q: %v", packagePattern, err)
		}

		if n := len(rawPackages); n != 1 {
			var err error

			if n == 0 {
				err = fmt.Errorf("methodset: no package found; packagePattern=%q", packagePattern)
			} else {
				packagePaths := make([]string, n)

				for i, rawPackage := range rawPackages {
					packagePaths[i] = rawPackage.PkgPath
				}

				err = fmt.Errorf("methodset: multiple packages found; packagePattern=%q packagePaths=%q",
					packagePattern, packagePaths)
			}

			return nil, err
		}

		rawPackage = rawPackages[0]
		importPath = packagePathToImportPath(rawPackage.PkgPath)
	} else {
		importPath = importPathOrPackagePattern
		currentPackage := pc.packagePath2Package[currentPackagePath]

		if currentPackage.ImportPath == importPath {
			return currentPackage, nil
		}

		rawPackage = currentPackage.Imports[importPath]
	}

	package1, err := pc.doImportPackage1(rawPackage, importPath)

	if err != nil {
		return nil, err
	}

	if err := pc.doImportPackage2(package1); err != nil {
		return nil, err
	}

	return package1, nil
}

// PackageBasicInfo ...
type PackageBasicInfo struct {
	Name       string
	Path       string
	ImportPath string
}

func (pc *ParseContext) getImportPath(object *ast.Object) (string, bool) {
	if package1, ok := pc.object2Package[object]; ok {
		return package1.ImportPath, true
	}

	return "", false
}

func (pc *ParseContext) getPackageBasicInfo(object *ast.Object) (PackageBasicInfo, bool) {
	if package1, ok := pc.object2Package[object]; ok {
		return PackageBasicInfo{package1.Name, package1.PkgPath, package1.ImportPath}, true
	}

	return PackageBasicInfo{}, false
}

func (pc *ParseContext) doImportPackage1(rawPackage *packages.Package, importPath string) (*package1, error) {
	if len(rawPackage.Errors) != 0 {
		buffer := bytes.NewBufferString(fmt.Sprintf("methodset: package errors encountered; packagePath=%q", rawPackage.PkgPath))

		for _, packageError := range rawPackage.Errors {
			buffer.WriteByte('\n')
			buffer.WriteString(packageError.Error())
		}

		return nil, errors.New(buffer.String())
	}

	if package1, ok := pc.packagePath2Package[rawPackage.PkgPath]; ok {
		return package1, nil
	}

	packageScope := ast.NewScope(universe)

	for _, file := range rawPackage.Syntax {
		for _, object := range file.Scope.Objects {
			packageScope.Insert(object)
		}
	}

	package1 := package1{
		Package:    rawPackage,
		ImportPath: importPath,
		Scope:      packageScope,
	}

	pc.packagePath2Package[package1.PkgPath] = &package1

	for _, file := range package1.Syntax {
		for _, object := range file.Scope.Objects {
			pc.object2Package[object] = &package1
		}
	}

	return &package1, nil
}

func (pc *ParseContext) doImportPackage2(package1 *package1) error {
	if package1.IsImported {
		return nil
	}

	for _, file := range package1.Syntax {
		fileScope := ast.NewScope(package1.Scope)

		for _, importSpec := range file.Imports {
			importPath, _ := strconv.Unquote(importSpec.Path.Value)
			rawNextPackage := package1.Imports[importPath]
			nextPackage, err := pc.doImportPackage1(rawNextPackage, importPath)

			if err != nil {
				return err
			}

			var importName string

			if importSpec.Name == nil {
				importName = nextPackage.Name
			} else {
				importName = importSpec.Name.Name
			}

			switch importName {
			case "_":
			case ".":
				for _, object := range nextPackage.Scope.Objects {
					fileScope.Insert(object)
				}
			default:
				object := ast.NewObj(ast.Pkg, importName)
				object.Decl = importSpec
				object.Data = nextPackage.Scope
				fileScope.Insert(object)
			}
		}

		i := 0

		for _, ident := range file.Unresolved {
			if !resolveIdent(ident, fileScope) {
				file.Unresolved[i] = ident
				i++
			}
		}

		file.Unresolved = file.Unresolved[:i]
		var visitor ast.Visitor

		visitor = visitorFunc(func(node ast.Node) ast.Visitor {
			if node, ok := node.(*ast.SelectorExpr); ok {
				if x, ok := node.X.(*ast.Ident); ok && x.Obj != nil && x.Obj.Kind == ast.Pkg {
					packageScope := x.Obj.Data.(*ast.Scope)
					resolveIdent(node.Sel, packageScope)
				}

				return nil
			}

			return visitor
		})

		ast.Walk(visitor, file)
	}

	package1.IsImported = true
	return nil
}

type package1 struct {
	*packages.Package

	ImportPath string
	Scope      *ast.Scope
	IsImported bool
}

func packagePathToImportPath(packagePath string) string {
	if strings.HasPrefix(packagePath, "_/") {
		return packagePath
	}

	s := packagePath

	for {
		i := strings.Index(s, "vendor/")

		if i < 0 {
			return packagePath
		}

		if i >= 1 && s[i-1] != '/' {
			s = s[:i]
			continue
		}

		return packagePath[i+len("vendor/"):]
	}
}

func resolveIdent(ident *ast.Ident, scope *ast.Scope) bool {
	for {
		if object := scope.Lookup(ident.Name); object != nil {
			ident.Obj = object
			return true
		}

		scope = scope.Outer

		if scope == nil {
			return false
		}
	}
}

var buildContext = func() build.Context {
	buildContext := build.Default
	buildContext.CgoEnabled = false
	return buildContext
}()

var universe = func() *ast.Scope {
	universe := ast.NewScope(nil)

	for _, v := range []struct {
		ObjectName string
		ObjectKind ast.ObjKind
	}{
		// See https://golang.org/pkg/builtin/

		{"true", ast.Con},
		{"false", ast.Con},
		{"iota", ast.Con},

		{"nil", ast.Var},

		{"append", ast.Fun},
		{"cap", ast.Fun},
		{"close", ast.Fun},
		{"complex", ast.Fun},
		{"copy", ast.Fun},
		{"delete", ast.Fun},
		{"imag", ast.Fun},
		{"len", ast.Fun},
		{"make", ast.Fun},
		{"new", ast.Fun},
		{"panic", ast.Fun},
		{"print", ast.Fun},
		{"println", ast.Fun},
		{"real", ast.Fun},
		{"recover", ast.Fun},

		{"ComplexType", ast.Typ},
		{"FloatType", ast.Typ},
		{"IntegerType", ast.Typ},
		{"Type", ast.Typ},
		{"Type1", ast.Typ},
		{"bool", ast.Typ},
		{"byte", ast.Typ},
		{"complex128", ast.Typ},
		{"complex64", ast.Typ},
		{"error", ast.Typ},
		{"float32", ast.Typ},
		{"float64", ast.Typ},
		{"int", ast.Typ},
		{"int16", ast.Typ},
		{"int32", ast.Typ},
		{"int64", ast.Typ},
		{"int8", ast.Typ},
		{"rune", ast.Typ},
		{"string", ast.Typ},
		{"uint", ast.Typ},
		{"uint16", ast.Typ},
		{"uint32", ast.Typ},
		{"uint64", ast.Typ},
		{"uint8", ast.Typ},
		{"uintptr", ast.Typ},
	} {
		universe.Insert(ast.NewObj(v.ObjectKind, v.ObjectName))
	}

	return universe
}()
