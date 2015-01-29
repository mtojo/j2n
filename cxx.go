package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/mtojo/go-java/java"
)

var cxxTypes = map[string]string{
	"B":        "std::uint8_t",
	"C":        "char16_t",
	"D":        "double",
	"F":        "float",
	"I":        "std::int32_t",
	"J":        "std::int64_t",
	"LString;": "Ljava/lang/String;",
	"S":        "std::int16_t",
	"V":        "void",
	"X":        "Ljava/lang/String;",
	"Z":        "bool",
}

var cxxPrimitives = []string{
	"bool",
	"char",
	"char16_t",
	"char32_t",
	"double",
	"float",
	"int",
	"int16_t",
	"int32_t",
	"int64_t",
	"int8_t",
	"long int",
	"long long int",
	"long",
	"size_t",
	"uint16_t",
	"uint32_t",
	"uint64_t",
	"uint8_t",
	"unsigned char",
	"void",
	"wchar_t",
}

var cxxReservedKeywords = []string{
	"alignas",
	"alignof",
	"and",
	"and_eq",
	"asm",
	"auto",
	"bitand",
	"bitor",
	"bool",
	"break",
	"case",
	"catch",
	"char",
	"char16_t",
	"char32_t",
	"class",
	"compl",
	"const",
	"const_cast",
	"constexpr",
	"continue",
	"decltype",
	"default",
	"delete",
	"do",
	"double",
	"dynamic_cast",
	"else",
	"enum",
	"explicit",
	"export",
	"extern",
	"false",
	"final",
	"float",
	"for",
	"friend",
	"goto",
	"if",
	"inline",
	"int",
	"long",
	"mutable",
	"namespace",
	"new",
	"noexcept",
	"not",
	"not_eq",
	"nullptr",
	"operator",
	"or",
	"or_eq",
	"override",
	"private",
	"protected",
	"public",
	"register",
	"reinterpret_cast",
	"return",
	"short",
	"signed",
	"sizeof",
	"static",
	"static_assert",
	"static_cast",
	"struct",
	"switch",
	"template",
	"this",
	"thread_local",
	"throw",
	"true",
	"try",
	"typedef",
	"typeid",
	"typename",
	"union",
	"unsigned",
	"using",
	"virtual",
	"void",
	"volatile",
	"wchar_t",
	"while",
	"xor",
	"xor_eq",
}

func isCxxPrimitive(s string) bool {
	re, _ := regexp.Compile("^(?:(?:un)?signed|std::)\\s*")
	typ := re.ReplaceAllString(s, "")
	for _, t := range cxxPrimitives {
		if t == typ {
			return true
		}
	}
	return false
}

func isCxxReserved(s string) bool {
	for _, t := range cxxReservedKeywords {
		if t == s {
			return true
		}
	}
	return false
}

func parseMethodSignature(signature string) (string, []string) {
	ret := signature[strings.Index(signature, ")")+1:]

	var args []string
	for i := 1; i < strings.Index(signature, ")"); i++ {
		head := i
		for signature[i] == '[' {
			i++
		}
		if signature[i] == 'L' {
			pos := strings.Index(signature[i:], ";")
			i += pos
		}
		tail := i + 1
		argumentTypeSignature := signature[head:tail]
		args = append(args, argumentTypeSignature)
	}

	return ret, args
}

func lookupCxxType(jType string) string {
	if t, ok := cxxTypes[jType]; ok {
		jType = t
	}

	if jType[0] == '[' {
		return "std::vector<" + lookupCxxType(jType[1:]) + ">"
	}

	if jType[0] == 'L' {
		jType = jType[1 : len(jType)-1]
		jType = strings.Replace(jType, "$", "_", -1)
		s := []string{}
		if *namespacePrefix != "" {
			for _, elem := range strings.Split(*namespacePrefix, "::") {
				if isCxxReserved(elem) {
					elem += "_"
				}
				s = append(s, elem)
			}
		}
		for _, elem := range strings.Split(jType, "/") {
			if isCxxReserved(elem) {
				elem += "_"
			}
			s = append(s, elem)
		}
		return strings.Join(s, "::")
	}

	return jType
}

func lookupCxxHeader(jType string) *string {
	if t, ok := cxxTypes[jType]; ok {
		jType = t
	}

	if jType[0] == '[' {
		return lookupCxxHeader(jType[1:])
	}

	if jType[0] == 'L' {
		s := strings.Replace(jType[1:len(jType)-1], "$", "_", -1)
		if *headerExt != "" {
			s += "." + *headerExt
		}
		return &s
	}

	return nil
}

func formatCxxFile(fname string) error {
	if *clangFormatFile == "" {
		return nil
	}

	cmd := exec.Command("clang-format", "-i",
		"-assume-filename=\""+*clangFormatFile+"\"", fname)
	return cmd.Run()
}

func quoteCxxString(str string) string {
	return strconv.Quote(str)
}

func formatCxxConstant(c *java.ClassFile, attr *java.ConstantValueAttribute) *string {
	if attr != nil {
		constant := c.ConstantPool[attr.ConstantValueIndex-1]

		switch constant.GetTag() {
		case java.IntegerConstant:
			v := fmt.Sprint(constant.Integer().Value)
			return &v
		case java.LongConstant:
			v := fmt.Sprint(constant.Long().Value)
			return &v
		case java.FloatConstant:
			v := fmt.Sprint(constant.Float().Value)
			if v == "NaN" {
				v = "std::numeric_limits<float>::quiet_NaN()"
			} else if v == "+Inf" {
				v = "+std::numeric_limits<float>::infinity()"
			} else if v == "-Inf" {
				v = "-std::numeric_limits<float>::infinity()"
			}
			return &v
		case java.DoubleConstant:
			v := fmt.Sprint(constant.Double().Value)
			if v == "NaN" {
				v = "std::numeric_limits<double>::quiet_NaN()"
			} else if v == "+Inf" {
				v = "+std::numeric_limits<double>::infinity()"
			} else if v == "-Inf" {
				v = "-std::numeric_limits<double>::infinity()"
			}
			return &v
		case java.Utf8Constant:
			v := quoteCxxString(constant.Utf8().Value)
			return &v
		case java.StringConstant:
			v := quoteCxxString(
				c.ConstantPool[constant.String().StringIndex-1].Utf8().Value)
			return &v
		}
	}

	return nil
}
