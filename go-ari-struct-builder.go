package main

import (
	"flag"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"unicode"
	"strings"
	"log"
)

var (
	tmpgostructs map[string]GoStruct
	gostructs map[string]GoStruct
)

type Models struct {
	JSON interface{} `json:"models"`
}

type Field struct {
	Name string
	Type string
	JSONName string
}
type GoStruct struct {
	Name string
	Fields []Field
	SubTypes []string
	Parent string
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
func BuildField(f Field, m map[string]interface{}) Field {
	for key, value := range m {
		switch key {
		case "type":
			var typestring string
			v := value.(string)
			if strings.HasPrefix(v, "List[") {
				typestring = strings.TrimPrefix(v, "List[")
				typestring = strings.TrimSuffix(typestring, "]")
				typestring = strings.Join([]string{"[]", typestring}, "")
			} else if v == "object" {
				typestring = "string"
			} else if v == "long" {
				typestring = "uint64"
			} else if v == "double" {
				typestring = "float64"
			} else if v == "Date" {
				typestring = "string"
			} else if v== "boolean" {
				typestring = "bool"
			} else {
				typestring = v
			}
			f.Type = typestring
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
	modeldir := flag.String("models", "", "Path to model files")
	flag.Parse()
	files, err := ioutil.ReadDir(*modeldir)
	if err != nil {
		log.Fatal(err)
	}
	for _, modelfile := range files {
		if !modelfile.IsDir() {
			modelpath := strings.Join([]string{*modeldir, modelfile.Name()}, "/")
			modelstring, err := ioutil.ReadFile(modelpath)
			if err != nil {
				continue
			}
			var m Models
			json.Unmarshal(modelstring, &m)
			ParseModels(m.JSON.(map[string]interface{}))
		}
	}

	fmt.Println("package nv\n")
	BuildConstructors()
	BuildVar()
	BuildInit()
	OutputStructs()
}