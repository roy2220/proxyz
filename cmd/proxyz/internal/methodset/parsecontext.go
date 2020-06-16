package methodset

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"strconv"

	"errors"
	"golang.org/x/tools/go/packages"
)

// ParseContext ...
type ParseContext struct {
	fileSet           *token.FileSet
	packageID2Package map[string]*package1
	object2Package    map[*ast.Object]*package1
}

// Init ...
func (pc *ParseContext) Init() *ParseContext {
	pc.fileSet = token.NewFileSet()
	pc.packageID2Package = map[string]*package1{}
	pc.object2Package = map[*ast.Object]*package1{}
	return pc
}

// FileSet ...
func (pc *ParseContext) FileSet() *token.FileSet {
	return pc.fileSet
}

func (pc *ParseContext) importPackage(currentPackageID string, packagePath string) (*package1, error) {
	var rawPackage *packages.Package

	if currentPackageID == "" {
		rawPackages, err := packages.Load(&packages.Config{
			Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedSyntax,
			Fset: pc.fileSet,
		}, packagePath)

		if err != nil {
			return nil, fmt.Errorf("methodset: package load failed; packagePath=%q: %v", packagePath, err)
		}

		if n := len(rawPackages); n != 1 {
			var err error

			if n == 0 {
				err = fmt.Errorf("methodset: no package found; packagePath=%q", packagePath)
			} else {
				err = fmt.Errorf("methodset: multiple packages found; packagePath=%q", packagePath)
			}

			return nil, err
		}

		rawPackage = rawPackages[0]
	} else {
		currentPackage := pc.packageID2Package[currentPackageID]
		rawPackage = currentPackage.Imports[packagePath]
	}

	package1, err := pc.doImportPackage1(rawPackage, packagePath)

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
	ID   string
	Name string
	Path string
}

func (pc *ParseContext) getPackageBasicInfo(object *ast.Object) (PackageBasicInfo, bool) {
	if package1, ok := pc.object2Package[object]; ok {
		return PackageBasicInfo{package1.ID, package1.Name, package1.Path}, true
	}

	return PackageBasicInfo{}, false
}

func (pc *ParseContext) doImportPackage1(rawPackage *packages.Package, packagePath string) (*package1, error) {
	if len(rawPackage.Errors) != 0 {
		buffer := bytes.NewBufferString("methodset: package errors occurred")

		for _, packageError := range rawPackage.Errors {
			buffer.WriteByte('\n')
			buffer.WriteString(packageError.Error())
		}

		return nil, errors.New(buffer.String())
	}

	if package1, ok := pc.packageID2Package[rawPackage.ID]; ok {
		return package1, nil
	}

	scope := ast.NewScope(universe)

	for _, file := range rawPackage.Syntax {
		for _, object := range file.Scope.Objects {
			scope.Insert(object)
		}
	}

	package1 := package1{
		Package: rawPackage,
		Path:    packagePath,
		Scope:   scope,
	}

	pc.packageID2Package[package1.ID] = &package1

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
			dependedPackagePath, _ := strconv.Unquote(importSpec.Path.Value)
			rawDependedPackage := package1.Imports[dependedPackagePath]
			dependedPackage, err := pc.doImportPackage1(rawDependedPackage, dependedPackagePath)

			if err != nil {
				return err
			}

			var importName string

			if importSpec.Name == nil {
				importName = dependedPackage.Name
			} else {
				importName = importSpec.Name.Name
			}

			switch importName {
			case "_":
			case ".":
				for _, object := range dependedPackage.Scope.Objects {
					fileScope.Insert(object)
				}
			default:
				object := ast.NewObj(ast.Pkg, importName)
				object.Decl = importSpec
				object.Data = dependedPackage.Scope
				fileScope.Insert(object)
				pc.object2Package[object] = dependedPackage
			}
		}

		for _, ident := range file.Unresolved {
			if !resolveIdent(ident, fileScope) {
				return fmt.Errorf("methodset: undeclared name; name=%q sourcePosition=%q",
					ident.Name, pc.fileSet.Position(ident.Pos()))
			}
		}
	}

	package1.IsImported = true
	return nil
}

type package1 struct {
	*packages.Package

	Path       string
	Scope      *ast.Scope
	IsImported bool
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
