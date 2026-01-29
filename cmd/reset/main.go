// Package main provides a utility for generating Reset() methods for Go structs.
// Structs marked with // generate:reset comment will have Reset() methods generated.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// FieldKind represents the category of a field type.
type FieldKind int

const (
	KindUnknown FieldKind = iota
	KindInt
	KindFloat
	KindComplex
	KindBool
	KindString
	KindSlice
	KindMap
	KindStruct
	KindInterface
	KindFunc
	KindChan
	KindPointer
	KindArray
)

// FieldInfo holds information about a struct field.
type FieldInfo struct {
	Name       string    // Field name
	TypeString string    // String representation of type
	Kind       FieldKind // Type category
	ElemKind   FieldKind // For pointers - element type kind
	ElemType   string    // For pointers - element type string
	IsPointer  bool      // Is this a pointer type
	HasReset   bool      // Does the type have a Reset() method
}

// StructInfo holds information about a struct to generate Reset() for.
type StructInfo struct {
	Name     string      // Struct name
	RecvName string      // Receiver name (first letter lowercase)
	Fields   []FieldInfo // Struct fields
	Package  string      // Package name
	FilePath string      // Source file path
}

var (
	dir     = flag.String("dir", ".", "Directory to scan for packages")
	verbose = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	if *verbose {
		log.Printf("Scanning directory: %s", absDir)
	}

	pkgs, err := loadPackages(absDir)
	if err != nil {
		log.Fatalf("Failed to load packages: %v", err)
	}

	if *verbose {
		log.Printf("Loaded %d packages", len(pkgs))
	}

	// Group structs by directory for generating files
	structsByDir := make(map[string][]StructInfo)

	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}

		structs := findMarkedStructs(pkg)
		for _, s := range structs {
			dir := filepath.Dir(s.FilePath)
			structsByDir[dir] = append(structsByDir[dir], s)
		}
	}

	if len(structsByDir) == 0 {
		if *verbose {
			log.Println("No structs with // generate:reset marker found")
		}
		return
	}

	for dir, structs := range structsByDir {
		if err := writeGeneratedFile(dir, structs); err != nil {
			log.Printf("Failed to write generated file in %s: %v", dir, err)
		} else if *verbose {
			log.Printf("Generated reset.gen.go in %s with %d struct(s)", dir, len(structs))
		}
	}
}

// loadPackages loads all packages in the given directory recursively.
func loadPackages(dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax,
		Dir: dir,
	}

	return packages.Load(cfg, "./...")
}

// findMarkedStructs finds all structs with // generate:reset comment in a package.
func findMarkedStructs(pkg *packages.Package) []StructInfo {
	var structs []StructInfo

	for i, file := range pkg.Syntax {
		filePath := ""
		if i < len(pkg.CompiledGoFiles) {
			filePath = pkg.CompiledGoFiles[i]
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// Check for // generate:reset comment
				if hasGenerateResetComment(genDecl, typeSpec, pkg.Fset) {
					info := extractStructInfo(pkg, typeSpec.Name.Name, structType, filePath)
					structs = append(structs, info)
					if *verbose {
						log.Printf("Found marked struct: %s in %s", info.Name, filePath)
					}
				}
			}
		}
	}

	return structs
}

// hasGenerateResetComment checks if a type declaration has the // generate:reset comment.
func hasGenerateResetComment(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, fset *token.FileSet) bool {
	// Check GenDecl doc comments
	if genDecl.Doc != nil {
		for _, comment := range genDecl.Doc.List {
			if strings.Contains(comment.Text, "generate:reset") {
				return true
			}
		}
	}

	// Check TypeSpec doc comments
	if typeSpec.Doc != nil {
		for _, comment := range typeSpec.Doc.List {
			if strings.Contains(comment.Text, "generate:reset") {
				return true
			}
		}
	}

	// Check TypeSpec comment (inline comment)
	if typeSpec.Comment != nil {
		for _, comment := range typeSpec.Comment.List {
			if strings.Contains(comment.Text, "generate:reset") {
				return true
			}
		}
	}

	return false
}

// extractStructInfo extracts information about a struct's fields.
func extractStructInfo(pkg *packages.Package, name string, structType *ast.StructType, filePath string) StructInfo {
	info := StructInfo{
		Name:     name,
		RecvName: strings.ToLower(name[:1]),
		Package:  pkg.Name,
		FilePath: filePath,
	}

	// Get the types.Struct for this struct
	obj := pkg.Types.Scope().Lookup(name)
	if obj == nil {
		return info
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return info
	}

	underlying, ok := named.Underlying().(*types.Struct)
	if !ok {
		return info
	}

	for i := 0; i < underlying.NumFields(); i++ {
		field := underlying.Field(i)
		fieldInfo := analyzeField(field, pkg)
		info.Fields = append(info.Fields, fieldInfo)
	}

	return info
}

// analyzeField analyzes a struct field and returns its info.
func analyzeField(field *types.Var, pkg *packages.Package) FieldInfo {
	t := field.Type()
	info := FieldInfo{
		Name:       field.Name(),
		TypeString: types.TypeString(t, relativeTo(pkg.Types)),
	}

	info.Kind = classifyType(t)
	info.IsPointer = info.Kind == KindPointer

	if ptr, ok := t.(*types.Pointer); ok {
		elem := ptr.Elem()
		info.ElemKind = classifyType(elem)
		info.ElemType = types.TypeString(elem, relativeTo(pkg.Types))
		info.HasReset = hasResetMethod(types.NewPointer(elem))
	} else {
		info.HasReset = hasResetMethod(types.NewPointer(t))
	}

	return info
}

// relativeTo returns a qualifier function for types.TypeString.
func relativeTo(pkg *types.Package) types.Qualifier {
	return func(other *types.Package) string {
		if pkg == other {
			return ""
		}
		return other.Name()
	}
}

// classifyType determines the category of a type.
func classifyType(t types.Type) FieldKind {
	switch t := t.Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			return KindBool
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
			return KindInt
		case types.Float32, types.Float64:
			return KindFloat
		case types.Complex64, types.Complex128:
			return KindComplex
		case types.String:
			return KindString
		}
	case *types.Slice:
		return KindSlice
	case *types.Map:
		return KindMap
	case *types.Struct:
		return KindStruct
	case *types.Interface:
		return KindInterface
	case *types.Signature:
		return KindFunc
	case *types.Chan:
		return KindChan
	case *types.Pointer:
		return KindPointer
	case *types.Array:
		return KindArray
	}
	return KindUnknown
}

// hasResetMethod checks if a type has a Reset() method.
func hasResetMethod(t types.Type) bool {
	ms := types.NewMethodSet(t)
	for i := 0; i < ms.Len(); i++ {
		if ms.At(i).Obj().Name() == "Reset" {
			sig, ok := ms.At(i).Obj().Type().(*types.Signature)
			if ok && sig.Params().Len() == 0 && sig.Results().Len() == 0 {
				return true
			}
		}
	}
	return false
}

// generateResetCode generates the Reset() method code for a struct.
func generateResetCode(info StructInfo) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "// Reset resets all fields of %s to their zero values.\n", info.Name)
	fmt.Fprintf(&buf, "func (%s *%s) Reset() {\n", info.RecvName, info.Name)
	fmt.Fprintf(&buf, "\tif %s == nil {\n", info.RecvName)
	fmt.Fprintf(&buf, "\t\treturn\n")
	fmt.Fprintf(&buf, "\t}\n")

	for _, field := range info.Fields {
		code := generateFieldReset(info.RecvName, field)
		if code != "" {
			buf.WriteString(code)
		}
	}

	buf.WriteString("}\n")
	return buf.String()
}

// generateFieldReset generates the reset code for a single field.
func generateFieldReset(recv string, field FieldInfo) string {
	fieldRef := fmt.Sprintf("%s.%s", recv, field.Name)

	// Skip unexported fields from other packages (shouldn't happen in our case)
	if field.Name == "" || !isExportedOrLocal(field.Name) {
		// For unexported fields in the same package, we can still access them
		if field.Name == "" {
			return ""
		}
	}

	switch field.Kind {
	case KindInt, KindFloat, KindComplex:
		return fmt.Sprintf("\t%s = 0\n", fieldRef)

	case KindBool:
		return fmt.Sprintf("\t%s = false\n", fieldRef)

	case KindString:
		return fmt.Sprintf("\t%s = \"\"\n", fieldRef)

	case KindSlice:
		return fmt.Sprintf("\t%s = %s[:0]\n", fieldRef, fieldRef)

	case KindMap:
		return fmt.Sprintf("\tclear(%s)\n", fieldRef)

	case KindArray:
		return fmt.Sprintf("\t%s = %s{}\n", fieldRef, field.TypeString)

	case KindStruct:
		if field.HasReset {
			return fmt.Sprintf("\t%s.Reset()\n", fieldRef)
		}
		return fmt.Sprintf("\t%s = %s{}\n", fieldRef, field.TypeString)

	case KindInterface, KindFunc:
		return fmt.Sprintf("\t%s = nil\n", fieldRef)

	case KindChan:
		return fmt.Sprintf("\t// %s is a channel and is not reset\n", fieldRef)

	case KindPointer:
		return generatePointerFieldReset(recv, field)

	default:
		return fmt.Sprintf("\t// %s has unknown type and is not reset\n", fieldRef)
	}
}

// generatePointerFieldReset generates reset code for pointer fields.
func generatePointerFieldReset(recv string, field FieldInfo) string {
	fieldRef := fmt.Sprintf("%s.%s", recv, field.Name)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "\tif %s != nil {\n", fieldRef)

	switch field.ElemKind {
	case KindInt, KindFloat, KindComplex:
		fmt.Fprintf(&buf, "\t\t*%s = 0\n", fieldRef)

	case KindBool:
		fmt.Fprintf(&buf, "\t\t*%s = false\n", fieldRef)

	case KindString:
		fmt.Fprintf(&buf, "\t\t*%s = \"\"\n", fieldRef)

	case KindSlice:
		fmt.Fprintf(&buf, "\t\t*%s = (*%s)[:0]\n", fieldRef, fieldRef)

	case KindMap:
		fmt.Fprintf(&buf, "\t\tclear(*%s)\n", fieldRef)

	case KindArray:
		fmt.Fprintf(&buf, "\t\t*%s = %s{}\n", fieldRef, field.ElemType)

	case KindStruct:
		if field.HasReset {
			fmt.Fprintf(&buf, "\t\t%s.Reset()\n", fieldRef)
		} else {
			fmt.Fprintf(&buf, "\t\t*%s = %s{}\n", fieldRef, field.ElemType)
		}

	case KindInterface, KindFunc:
		fmt.Fprintf(&buf, "\t\t*%s = nil\n", fieldRef)

	case KindChan:
		fmt.Fprintf(&buf, "\t\t// *%s is a channel and is not reset\n", fieldRef)

	default:
		fmt.Fprintf(&buf, "\t\t// *%s has unknown type and is not reset\n", fieldRef)
	}

	buf.WriteString("\t}\n")
	return buf.String()
}

// isExportedOrLocal checks if a field name is exported or is a local unexported field.
func isExportedOrLocal(name string) bool {
	if name == "" {
		return false
	}
	// We handle both exported and unexported fields since we generate code in the same package
	return true
}

// writeGeneratedFile writes the generated reset.gen.go file.
func writeGeneratedFile(dir string, structs []StructInfo) error {
	if len(structs) == 0 {
		return nil
	}

	// All structs in the same directory should have the same package name
	pkgName := structs[0].Package

	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by cmd/reset; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)

	// Write Reset methods
	for i, s := range structs {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(generateResetCode(s))
	}

	// Format the code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, write unformatted code for debugging
		formatted = buf.Bytes()
		log.Printf("Warning: failed to format generated code: %v", err)
	}

	// Write to file
	filePath := filepath.Join(dir, "reset.gen.go")
	return os.WriteFile(filePath, formatted, 0644)
}
