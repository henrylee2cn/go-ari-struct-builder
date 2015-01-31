package main

import (
	"fmt"
)

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
			v := value.(string)
			f.Type = convertType(v)
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
