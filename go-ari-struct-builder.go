package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"unicode"
)

var (
	tmpgostructs map[string]GoStruct
	gostructs    map[string]GoStruct
)

type Swagger struct {
	APIVersion string      `json:"apiVersion"`
	BasePath   string      `json:"basePath"`
	APIs       []API       `json:"apis"`
	Models     interface{} `json:"models"`
}

type API struct {
	Path        string      `json:"path"`
	Description string      `json:"description"`
	Operations  []Operation `json:"operations"`
}

type Operation struct {
	HTTPMethod    string      `json:"httpMethod"`
	Summary       string      `json:"summary"`
	Notes         string      `json:"notes"`
	Nickname      string      `json:"nickname"`
	ResponseClass string      `json:"responseClass"`
	Parameters    []Parameter `json:"parameters"`
}

type Parameter struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ParamType     string `json:"paramType"`
	Required      bool   `json:"required"`
	AllowMultiple bool   `json:"allowMultiple"`
	Datatype      string `json:"dataType"`
}

type Models struct {
	JSON interface{} `json:"models"`
}

type Field struct {
	Name     string
	Type     string
	JSONName string
}
type GoStruct struct {
	Name     string
	Fields   []Field
	SubTypes []string
	Parent   string
}

func Canonicalize(s string) string {
	a := []rune(s)
	a[0] = unicode.ToUpper(a[0])
	for index, _ := range a {
		if index == 0 {
			a[0] = unicode.ToUpper(a[0])
			continue
		}
		if a[index-1] == '_' {
			a[index] = unicode.ToUpper(a[index])
		}
		if a[index] == 'I' && a[index+1] == 'd' {
			a[index+1] = unicode.ToUpper(a[index+1])
		}
	}
	return string(a)
}

func ParseModels(m map[string]interface{}) {
	for key, value := range m {
		s := GoStruct{Name: key, Parent: ""}
		v := value.(map[string]interface{})
		tmpgostructs[key] = BuildStruct(s, v)
	}
	for _, t := range tmpgostructs {
		for _, st := range t.SubTypes {
			sub := tmpgostructs[st]
			sub.Parent = t.Name
			tmpgostructs[st] = sub
		}
	}
	for ParentsExist() {
		ProcessSubTypes()
	}
	for _, t := range tmpgostructs {
		gostructs[t.Name] = t
	}
}

func ParentsExist() bool {
	for _, t := range tmpgostructs {
		if t.Parent != "" {
			return true
		}
	}
	return false
}

func ProcessSubTypes() {
	for _, t := range tmpgostructs {
		if t.Parent == "" {
			for _, subtype := range t.SubTypes {
				st := tmpgostructs[subtype]
				parent := tmpgostructs[st.Parent]
				for _, f := range parent.Fields {
					st.Fields = append(st.Fields, f)
				}
				st.Parent = ""
				tmpgostructs[subtype] = st
			}
			gostructs[t.Name] = t
			delete(tmpgostructs, t.Name)
		}
	}
}
func BuildStruct(s GoStruct, m map[string]interface{}) GoStruct {
	for key, value := range m {
		switch value.(type) {
		case string:
			continue
		case []interface{}:
			switch key {
			case "subTypes":
				v := value.([]interface{})
				s.SubTypes = BuildSubTypes(v)
			}
		case interface{}:
			v := value.(map[string]interface{})
			switch key {
			case "properties":
				s.Fields = BuildFields(v)
			}
		}
	}
	return s
}

func BuildSubTypes(st []interface{}) []string {
	subtypes := make([]string, 0, 50)
	for _, value := range st {
		subtypes = append(subtypes, value.(string))
	}
	return subtypes
}

func BuildFields(m map[string]interface{}) []Field {
	fields := make([]Field, 0, 5)
	for key, value := range m {
		f := Field{Name: Canonicalize(key)}
		v := value.(map[string]interface{})
		f = BuildField(f, v)
		f.JSONName = key
		fields = append(fields, f)
	}
	return fields
}

func ConvertType(s string) string {
	if strings.HasPrefix(s, "List[") {
		typestring := strings.TrimPrefix(s, "List[")
		typestring = strings.TrimSuffix(typestring, "]")
		typestring = strings.Join([]string{"[]", typestring}, "")
		return typestring
	} else if s == "object" {
		return "string"
	} else if s == "long" {
		return "uint64"
	} else if s == "double" {
		return "float64"
	} else if s == "Date" {
		return "string"
	} else if s == "boolean" {
		return "bool"
	} else {
		return s
	}
}
func BuildField(f Field, m map[string]interface{}) Field {
	for key, value := range m {
		switch key {
		case "type":
			v := value.(string)
			f.Type = ConvertType(v)
		}
	}
	return f
}

func OutputStructs() {
	for _, s := range gostructs {
		fmt.Printf("type %s struct {\n", s.Name)
		for _, field := range s.Fields {
			fmt.Printf("	%s %s `json:\"%s\"`\n", field.Name, field.Type, field.JSONName)
		}
		fmt.Println("}\n")
	}
}

func BuildConstructors() {
	for _, s := range gostructs {
		fmt.Printf("func New%s() interface{} {\n", s.Name)
		fmt.Printf("	return %s{}\n", s.Name)
		fmt.Printf("}\n\n")
	}
}

func BuildVar() {
	fmt.Printf("var (\n")
	fmt.Printf("	NewStruct map[string]func() interface{}\n")
	fmt.Printf(")\n\n")
}
func BuildInit() {
	fmt.Printf("func init() {\n")
	fmt.Printf("NewStruct = make(map[string]func() interface{})\n")
	for _, s := range gostructs {
		fmt.Printf("	NewStruct[\"%s\"] = New%s\n", s.Name, s.Name)
	}
	fmt.Printf("}\n\n")
}

func init() {
	gostructs = make(map[string]GoStruct)
	tmpgostructs = make(map[string]GoStruct)
}
func main() {
	swaggerdir := flag.String("path", "", "Path to model files")
	buildStructs := flag.Bool("structs", true, "Whether or not to build structs")
	buildAPI := flag.Bool("api", true, "Whether or not to build the API")
	flag.Parse()
	files, err := ioutil.ReadDir(*swaggerdir)
	if err != nil {
		log.Fatal(err)
	}
	for _, swaggerfile := range files {
		if !swaggerfile.IsDir() {
			swaggerpath := strings.Join([]string{*swaggerdir, swaggerfile.Name()}, "/")
			swaggerstring, err := ioutil.ReadFile(swaggerpath)
			if err != nil {
				continue
			}
			//fmt.Println(string(swaggerstring))
			var s Swagger
			json.Unmarshal(swaggerstring, &s)
			ParseModels(s.Models.(map[string]interface{}))
		}
	}

	fmt.Println("package nv\n")
	if *buildStructs {
		BuildConstructors()
		BuildVar()
		BuildInit()
		OutputStructs()
	}

	if *buildAPI {
		fmt.Println("API stuff here\n")
	}
}
