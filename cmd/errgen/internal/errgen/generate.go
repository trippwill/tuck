package errgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const apperrImport = "github.com/trippwill/tuck/internal/apperr"

type Options struct {
	TypeNames      []string
	ConstraintName string
	Dir            string
	Output         string
}

func Generate(options Options) ([]byte, error) {
	if len(options.TypeNames) == 0 {
		return nil, errors.New("-types is required")
	}
	typeNames, err := normalizeTypeNames(options.TypeNames)
	if err != nil {
		return nil, err
	}
	dir := options.Dir
	if dir == "" {
		dir = "."
	}

	info, err := inspectPackage(dir, typeNames)
	if err != nil {
		return nil, err
	}
	constraintName := options.ConstraintName
	if constraintName == "" {
		constraintName = exportedName(info.PackageName) + "Err"
	}

	source, err := render(info.PackageName, typeNames, constraintName)
	if err != nil {
		return nil, err
	}
	if options.Output != "" {
		outputPath := options.Output
		if !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(dir, outputPath)
		}
		if err := os.WriteFile(outputPath, source, 0o644); err != nil {
			return nil, fmt.Errorf("write output %q: %w", outputPath, err)
		}
	}
	return source, nil
}

type packageInfo struct {
	PackageName string
}

func normalizeTypeNames(typeNames []string) ([]string, error) {
	normalized := make([]string, 0, len(typeNames))
	seen := make(map[string]struct{}, len(typeNames))
	for _, typeName := range typeNames {
		typeName = strings.TrimSpace(typeName)
		if typeName == "" {
			continue
		}
		if _, ok := seen[typeName]; ok {
			return nil, fmt.Errorf("duplicate type %q", typeName)
		}
		seen[typeName] = struct{}{}
		normalized = append(normalized, typeName)
	}
	if len(normalized) == 0 {
		return nil, errors.New("-types is required")
	}
	return normalized, nil
}

func inspectPackage(dir string, typeNames []string) (packageInfo, error) {
	files, err := parseFiles(dir)
	if err != nil {
		return packageInfo{}, err
	}
	if len(files) == 0 {
		return packageInfo{}, fmt.Errorf("no Go files found in %q", dir)
	}

	packageName := files[0].Name.Name
	selected := make(map[string]struct{}, len(typeNames))
	typeFound := make(map[string]bool, len(typeNames))
	stringType := make(map[string]bool, len(typeNames))
	constFound := make(map[string]bool, len(typeNames))
	for _, typeName := range typeNames {
		selected[typeName] = struct{}{}
	}

	for _, file := range files {
		if file.Name.Name != packageName {
			return packageInfo{}, fmt.Errorf("multiple packages found in %q", dir)
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			switch gen.Tok {
			case token.TYPE:
				for _, spec := range gen.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					typeName := typeSpec.Name.Name
					if _, ok := selected[typeName]; !ok {
						continue
					}
					typeFound[typeName] = true
					if ident, ok := typeSpec.Type.(*ast.Ident); ok && ident.Name == "string" {
						stringType[typeName] = true
					}
				}
			case token.CONST:
				previousType := ""
				for _, spec := range gen.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						previousType = ""
						continue
					}
					effectiveType := identName(valueSpec.Type)
					if valueSpec.Type == nil {
						if len(valueSpec.Values) == 0 {
							effectiveType = previousType
						} else {
							effectiveType = ""
						}
					}
					if _, ok := selected[effectiveType]; ok {
						constFound[effectiveType] = true
					}
					previousType = effectiveType
				}
			}
		}
	}

	for _, typeName := range typeNames {
		if !typeFound[typeName] {
			return packageInfo{}, fmt.Errorf("type %q not found", typeName)
		}
		if !stringType[typeName] {
			return packageInfo{}, fmt.Errorf("type %q must have underlying type string", typeName)
		}
		if !constFound[typeName] {
			return packageInfo{}, fmt.Errorf("no constants of type %q found", typeName)
		}
	}

	return packageInfo{PackageName: packageName}, nil
}

func parseFiles(dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		if isGenerated(path) {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", path, err)
		}
		files = append(files, file)
	}
	return files, nil
}

func isGenerated(path string) bool {
	contents, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(contents), "\n") {
		if strings.Contains(line, "Code generated") && strings.Contains(line, "DO NOT EDIT") {
			return true
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			return false
		}
	}
	return false
}

func identName(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

func render(packageName string, typeNames []string, constraintName string) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by errgen; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", packageName)
	fmt.Fprintf(&b, "import %q\n\n", apperrImport)
	for _, typeName := range typeNames {
		fmt.Fprintf(&b, "func (k %s) Error() string { return string(k) }\n", typeName)
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "type %s interface {\n", constraintName)
	fmt.Fprintf(&b, "\terror\n")
	fmt.Fprintf(&b, "\tcomparable\n")
	fmt.Fprintf(&b, "\t%s\n", strings.Join(typeNames, " | "))
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "type Error[S %s] = apperr.Error[S]\n\n", constraintName)
	fmt.Fprintf(&b, "func AppErrMsg[S %s](sentinel S, msg string) error {\n", constraintName)
	fmt.Fprintf(&b, "\treturn apperr.AppErrMsg(sentinel, msg)\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func AppErrMsgf[S %s](sentinel S, format string, args ...any) error {\n", constraintName)
	fmt.Fprintf(&b, "\treturn apperr.AppErrMsgf(sentinel, format, args...)\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func AppErrWrap[S %s](sentinel S, err error) error {\n", constraintName)
	fmt.Fprintf(&b, "\treturn apperr.AppErrWrap(sentinel, err)\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func AppErrWrapf[S %s](sentinel S, err error, format string, args ...any) error {\n", constraintName)
	fmt.Fprintf(&b, "\treturn apperr.AppErrWrapf(sentinel, err, format, args...)\n")
	fmt.Fprintf(&b, "}\n")

	source, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w", err)
	}
	return source, nil
}

func exportedName(name string) string {
	var b strings.Builder
	capitalizeNext := true
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			capitalizeNext = true
			continue
		}
		if b.Len() == 0 && unicode.IsDigit(r) {
			b.WriteRune('_')
		}
		if capitalizeNext {
			b.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "Package"
	}
	return b.String()
}
