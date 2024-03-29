package main

import (
	_ "embed"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"text/template"
	"unicode"
)

//go:embed template.tmpl
var tmpl string

func main() {

	flag.Parse()

	for i := 0; i < flag.NArg(); i++ {

		inputFile := flag.Arg(i)
		outputFile := strings.Replace(inputFile, ".go", "_easystruct.go", 1)

		pf, err := parser.ParseFile(token.NewFileSet(), inputFile, nil, 0)
		if err != nil {
			panic(err)
		}

		data := fileData{
			Imports: map[string]struct{}{
				"net/http": {},
			},
		}

		ast.Inspect(pf, func(node ast.Node) bool {
			p, ok := node.(*ast.File)
			if ok {
				data.Package = p.Name.Name
				return true
			}

			ts, ok := node.(*ast.TypeSpec)
			if !ok || ts.Type == nil {
				return true
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			fields := make([]fieldData, 0, len(st.Fields.List))
		FIELDS:
			for _, item := range st.Fields.List {
				source, name, ok := matchTagValue("es", item)
				if !ok {
					continue
				}

				field := fieldData{
					Receiver: onlyUppers(ts.Name.Name),
					Error:    fmt.Sprintf("\"%s:%s: %%w\"", source, name),
				}

				switch source {
				case "query":
					field.Source = fmt.Sprintf("r.URL.Query().Get(%q)", name)
				case "header":
					field.Source = fmt.Sprintf("r.Header.Get(%q)", name)
				case "formData":
					field.Source = fmt.Sprintf("r.FormValue(%q)", name)
				default:
					continue FIELDS
				}

				switch ft := item.Type.(type) {
				case *ast.Ident:
					field.Type = ft.Name
				case *ast.ArrayType:
					if typ, ok := ft.Elt.(*ast.Ident); ok {
						field.Type = fmt.Sprintf("[]%s", typ.Name)
					}
				}

				switch field.Type {
				case "string", "[]byte", "[]rune":
					field.Kind = "varchar"
				case "[]int", "[]int8", "[]int16", "[]int32", "[]int64", "[]uint", "[]uint8", "[]uint16", "[]uint32", "[]uint64":
					field.Kind = "integers"
					field.Type = strings.TrimPrefix(field.Type, "[]")
					data.Import("fmt", "strings", "strconv")
				case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
					field.Kind = "integer"
					data.Import("fmt", "strconv")
				case "[]float64", "[]float32":
					field.Kind = "doubles"
					field.Type = strings.TrimPrefix(field.Type, "[]")
					data.Import("fmt", "strings", "strconv")
				case "float64", "float32":
					field.Kind = "double"
					data.Import("fmt", "strconv")
				case "bool":
					field.Kind = "boolean"
					data.Import("fmt", "strconv")
				case "[]bool":
					field.Kind = "booleans"
					field.Type = strings.TrimPrefix(field.Type, "[]")
					data.Import("fmt", "strings", "strconv")
				case "[]string":
					field.Kind = "strings"
					data.Import("strings")
				default:
					continue FIELDS
				}

				for _, name := range item.Names {
					if ok := name.IsExported(); ok {
						field.Name = name.Name
					}
				}
				if field.Name != "" {
					fields = append(fields, field)
				}
			}

			data.Structs = append(data.Structs, structData{
				Name:     ts.Name.Name,
				Receiver: onlyUppers(ts.Name.Name),
				Fields:   fields,
			})

			return true
		})

		if len(data.Structs) == 0 {
			continue
		}

		target, err := os.Create(outputFile)
		if err != nil {
			panic(err)
		}
		defer target.Close()

		if err := template.Must(template.New("").Parse(tmpl)).Execute(target, data); err != nil {
			panic(err)
		}
	}
}

type fileData struct {
	Package string
	Imports map[string]struct{}
	Structs []structData
}

func (fd *fileData) Import(pkgs ...string) {
	for _, pkg := range pkgs {
		if _, ok := fd.Imports[pkg]; !ok {
			fd.Imports[pkg] = struct{}{}
		}
	}
}

type structData struct {
	Name     string
	Receiver string
	Fields   []fieldData
}

type fieldData struct {
	Error        string
	Name, Type   string
	Source, Kind string
	Receiver     string
}

func onlyUppers(origin string) string {
	var rs []rune
	for _, r := range origin {
		if unicode.IsUpper(r) {
			rs = append(rs, r)
		}
	}
	return strings.ToLower(string(rs))
}

func matchTagValue(name string, field *ast.Field) (string, string, bool) {
	if field.Tag == nil || len(field.Tag.Value) == 0 {
		return "", "", false
	}
	tags := reflect.StructTag(field.Tag.Value[1:])
	tagValue, ok := tags.Lookup(name)
	if !ok {
		return "", "", false
	}
	return strings.Cut(tagValue, "=")
}
