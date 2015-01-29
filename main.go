package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/mtojo/go-java/java"
)

var (
	forceOutput        *bool
	jarFile            *string
	outDir             *string
	headerExt          *string
	sourceExt          *string
	incGuardPrefix     *string
	incGuardSuffix     *string
	namespacePrefix    *string
	headerTemplateFile *string
	sourceTemplateFile *string
	clangFormatFile    *string
)

func main() {
	// Configurations.
	jarFile = flag.String("i", "", "input jar file")
	outDir = flag.String("o", ".", "output directory")
	forceOutput = flag.Bool("f", false, "force output")
	headerExt = flag.String("x", "hpp", "header file extension")
	sourceExt = flag.String("c", "cpp", "source file extension")
	incGuardPrefix = flag.String("p", "", "include guard prefix")
	incGuardSuffix = flag.String("s", "", "include guard suffix")
	namespacePrefix = flag.String("n", "j2n", "namespace prefix")
	headerTemplateFile = flag.String("t", "", "header template file")
	sourceTemplateFile = flag.String("u", "", "source template file")
	clangFormatFile = flag.String("l", "", ".clang-format file")
	flag.Parse()

	if *jarFile == "" {
		fmt.Fprintln(os.Stderr, "no input jar file specified")
		os.Exit(1)
	}

	if _, err := os.Stat(*jarFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "input jar file is not exists: %s\n", *jarFile)
		os.Exit(1)
	}

	if *outDir == "" {
		*outDir = filepath.Dir(*jarFile)
	}

	if !*forceOutput {
		if _, err := os.Stat(*outDir); !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "output directory is already exists")
			os.Exit(1)
		}
	}

	// Make template.
	var header, source *template.Template
	if *headerTemplateFile != "" {
		buf, err := ioutil.ReadFile(*headerTemplateFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to read template file:", err)
			os.Exit(1)
		}
		header = makeTemplate("header", string(buf))

		if *sourceTemplateFile != "" {
			buf, err := ioutil.ReadFile(*sourceTemplateFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to read template file:", err)
				os.Exit(1)
			}
			source = makeTemplate("source", string(buf))
		}
	} else {
		buf, _ := Asset("assets/header.tmpl")
		header = makeTemplate("header", string(buf))
		buf, _ = Asset("assets/source.tmpl")
		source = makeTemplate("source", string(buf))
	}

	// Read android.jar file, and write C++ header files.
	err := java.ReadJarFile(*jarFile, func(fname string, c *java.ClassFile) {
		// Read class file data.
		fullName := c.GetClassName()
		packageName := fullName[0:strings.LastIndex(fullName, "/")]
		className := fullName[strings.LastIndex(fullName, "/")+1:]
		isFinal := (c.AccessFlags & java.FinalClass) > 0
		superClass := c.GetSuperClassName()

		data := ClassData{
			CommonData{
				fname,
				*headerExt,
				*incGuardPrefix,
				*incGuardSuffix,
				splitString(*namespacePrefix, "::"),
				cxxTypes,
			},
			fullName,
			packageName,
			className,
			isFinal,
			superClass,
			nil,
			nil,
			nil,
			nil,
			nil,
		}

		// Read fields data.
		for _, field := range c.Fields {
			if (field.AccessFlags & java.PublicField) == 0 {
				continue
			}

			name := c.GetFieldName(field)
			signature := c.GetFieldDescriptor(field)
			typ := lookupCxxType(signature)

			var constantValue *string
			for _, attr := range field.Attributes {
				constantValue = formatCxxConstant(c, attr.ConstantValueAttribute())
				if constantValue != nil {
					break
				}
			}

			if constantValue != nil {
				data.Constants = append(data.Constants, ConstantData{
					name,
					*constantValue,
					typ,
				})
			} else {
				if fname := lookupCxxHeader(signature); fname != nil {
					data.Dependencies = appendUnique(data.Dependencies, *fname)
				}

				isStatic := (field.AccessFlags & java.StaticField) > 0
				isEnum := (field.AccessFlags & java.EnumField) > 0
				isFinal := (field.AccessFlags & java.FinalField) > 0

				data.Fields = append(data.Fields, FieldData{
					name,
					signature,
					typ,
					isStatic,
					isEnum,
					isFinal,
				})
			}
		}

		// Read methods data.
	L:
		for _, method := range c.Methods {
			if (method.AccessFlags&java.PublicMethod) == 0 ||
				c.IsStaticInitializer(method) || c.IsNativeMethod(method) {
				continue
			}

			name := c.GetMethodName(method)
			signature := c.GetMethodDescriptor(method)
			ret, args := parseMethodSignature(signature)
			returnType := lookupCxxType(ret)
			if fname := lookupCxxHeader(ret); fname != nil {
				data.Dependencies = appendUnique(data.Dependencies, *fname)
			}
			var argumentTypes []string
			for _, arg := range args {
				argumentTypes = append(argumentTypes, lookupCxxType(arg))
				if fname := lookupCxxHeader(arg); fname != nil {
					data.Dependencies = appendUnique(data.Dependencies, *fname)
				}
			}

			// Prevent return type overloading.
			for _, m := range data.Methods {
				if name == m.Name && reflect.DeepEqual(argumentTypes, m.ArgumentTypes) {
					continue L
				}
			}

			isAbstractMethod := (method.AccessFlags & java.AbstractMethod) > 0
			isStaticMethod := (method.AccessFlags & java.StaticMethod) > 0

			methodData := MethodData{
				name,
				signature,
				returnType,
				argumentTypes,
				isAbstractMethod,
				isStaticMethod,
			}

			if c.IsInitializer(method) {
				data.Initializers = append(data.Initializers, methodData)
			} else {
				data.Methods = append(data.Methods, methodData)
			}
		}

		headerFile := *lookupCxxHeader("L" + fullName + ";")

		data.Dependencies = filter(data.Dependencies, func(fname string) bool {
			return fname != headerFile
		})

		sort.Sort(sort.StringSlice(data.Dependencies))

		// Write C++ header file.
		headerFile = path.Join(*outDir, headerFile)

		if !*forceOutput {
			if _, err := os.Stat(headerFile); !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "output file is already exists: %s", headerFile)
				os.Exit(1)
			}
		}

		if err := os.MkdirAll(filepath.Dir(headerFile), os.ModeDir|os.ModePerm); err != nil {
			fmt.Fprintln(os.Stderr, "failed to create output directory:", err)
			os.Exit(1)
		}

		{
			out, err := os.Create(headerFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create output file: %s\n", headerFile)
				os.Exit(1)
			}
			defer out.Close()

			if err := header.Execute(out, data); err != nil {
				fmt.Fprintln(os.Stderr, "failed to write output file:", err)
				os.Exit(1)
			}
		}

		if err := formatCxxFile(headerFile); err != nil {
			fmt.Fprintln(os.Stderr, "failed to execute clang-format command:", err)
			os.Exit(1)
		}

		// Write C++ source file.
		if source != nil {
			sourceFile := replaceExt(headerFile, *sourceExt)

			if !*forceOutput {
				if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "output file is already exists: %s", sourceFile)
					os.Exit(1)
				}
			}

			if err := os.MkdirAll(filepath.Dir(sourceFile), os.ModeDir|os.ModePerm); err != nil {
				fmt.Fprintln(os.Stderr, "failed to create output directory:", err)
				os.Exit(1)
			}

			{
				out, err := os.Create(sourceFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to create output file: %s\n", sourceFile)
					os.Exit(1)
				}
				defer out.Close()

				if err := source.Execute(out, data); err != nil {
					fmt.Fprintln(os.Stderr, "failed to write output file:", err)
					os.Exit(1)
				}
			}

			if err := formatCxxFile(sourceFile); err != nil {
				fmt.Fprintln(os.Stderr, "failed to execute clang-format command:", err)
				os.Exit(1)
			}
		}
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read jar file:", err)
		os.Exit(1)
	}

	// Write common C++ header file if default template is used.
	if *headerTemplateFile == "" {
		buf, _ := Asset("assets/helper.tmpl")
		tpl := makeTemplate("common", string(buf))
		outFile := *outDir + "/common." + *headerExt

		if !*forceOutput {
			if _, err := os.Stat(outFile); !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "output file is already exists: %s", outFile)
				os.Exit(1)
			}
		}

		{
			out, err := os.Create(outFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create output file: %s\n", outFile)
				os.Exit(1)
			}
			defer out.Close()

			data := CommonData{
				outFile,
				*headerExt,
				*incGuardPrefix,
				*incGuardSuffix,
				splitString(*namespacePrefix, "::"),
				cxxTypes,
			}

			if err := tpl.Execute(out, data); err != nil {
				fmt.Fprintln(os.Stderr, "failed to write output file:", err)
				os.Exit(1)
			}
		}

		if err := formatCxxFile(outFile); err != nil {
			fmt.Fprintln(os.Stderr, "failed to execute clang-format command:", err)
			os.Exit(1)
		}
	}
}
