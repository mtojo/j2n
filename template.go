package main

import (
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type CommonData struct {
	Filename        string
	HeaderExt       string
	MacroPrefix     string
	MacroSuffix     string
	NamespacePrefix []string
	TypeMap         map[string]string
}

type ConstantData struct {
	Name  string
	Value string
	Type  string
}

type FieldData struct {
	Name      string
	Signature string
	Type      string
	IsStatic  bool
	IsEnum    bool
	IsFinal   bool
}

type MethodData struct {
	Name          string
	Signature     string
	ReturnType    string
	ArgumentTypes []string
	IsAbstract    bool
	IsStatic      bool
}

type ClassData struct {
	CommonData
	FullName     string
	PackageName  string
	ClassName    string
	IsFinal      bool
	SuperClass   string
	Dependencies []string
	Constants    []ConstantData
	Initializers []MethodData
	Fields       []FieldData
	Methods      []MethodData
}

func makeTemplate(name, tpl string) *template.Template {
	t := template.New(name)
	t.Funcs(template.FuncMap{
		"Back": func(slice []string) string {
			return slice[len(slice)-1]
		},
		"Base": path.Base,
		"Dir": func(s string) string {
			dir := path.Dir(s)
			if dir == "." {
				return ""
			}
			return dir
		},
		"Ext": path.Ext,
		"Front": func(slice []string) string {
			return slice[0]
		},
		"IsPrimitive": isCxxPrimitive,
		"IsReserved":  isCxxReserved,
		"Join":        strings.Join,
		"LookupType": func(jType string) string {
			return lookupCxxType(jType)
		},
		"LookupHeader": func(jType string) string {
			if fname := lookupCxxHeader(jType); fname != nil {
				return *fname
			}
			return ""
		},
		"PopBack": func(slice []string) []string {
			return slice[:len(slice)-1]
		},
		"PopFront": func(slice []string) []string {
			return slice[1:]
		},
		"Replace": strings.Replace,
		"ReplaceAll": func(s, o, n string) string {
			return strings.Replace(s, o, n, -1)
		},
		"ReplaceExt": replaceExt,
		"Reverse": func(s []string) []string {
			for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
				s[i], s[j] = s[j], s[i]
			}
			return s
		},
		"SnakeCase": snakeCase,
		"Split":     splitString,
		"ToLower":   strings.ToLower,
		"ToUpper":   strings.ToUpper,
		"TrimExt": func(fname string) string {
			return strings.TrimSuffix(fname, filepath.Ext(fname))
		},
		"TrimNamespace": func(s string) string {
			return strings.TrimPrefix(s, *namespacePrefix+"::")
		},
		"TrimPrefix": func(s, prefix string) string {
			return strings.TrimPrefix(s, prefix)
		},
		"TrimSuffix": func(s, suffix string) string {
			return strings.TrimSuffix(s, suffix)
		},
		"UpperCamelCase": upperCamelCase,
	})
	template.Must(t.Parse(tpl))
	return t
}
