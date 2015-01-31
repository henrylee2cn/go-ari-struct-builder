package main

import (
	"bytes"
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
	clientAPIBuf *bytes.Buffer
)

var apiPreamble string = `import (
	"fmt"
	"bytes"
	"encoding/json"
	"errors"
)

func buildJSON(params map[string]string) string {
	mapsize := len(params)
	var counter int = 1
	body := bytes.NewBufferString("{")
	for key, value := range params {
		var s string
		if counter < mapsize {
			s = fmt.Sprintf("\"%s\":\"%s\",", key, value)
		} else {
			s = fmt.Sprintf("\"%s\":\"%s\"", key, value)
		}
		body.WriteString(s)
		counter++
	}
	body.WriteString("}")
	return body.String()
}
`

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
	HTTPMethod		string			`json:"httpMethod"`
	Summary			string    	  `json:"summary"`
	Notes         string      `json:"notes"`
	Nickname      string      `json:"nickname"`
	ResponseClass string      `json:"responseClass"`
	Parameters    []Parameter `json:"parameters"`
	ErrorResponses []ErrorReponse `json:"errorResponses"`
}

type ErrorReponse struct {
	Code	int		`json:"code"`
	Reason	string	`json:"reason"`
}
type Parameter struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ParamType     string `json:"paramType"`
	Required      bool   `json:"required"`
	AllowMultiple bool   `json:"allowMultiple"`
	DataType      string `json:"dataType"`
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

func convertType(s string) string {
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

func init() {
	gostructs = make(map[string]GoStruct)
	tmpgostructs = make(map[string]GoStruct)
	clientAPIBuf = bytes.NewBufferString("")
}
func main() {
	swaggerDir := flag.String("path", "", "Path to model files")
	buildStructs := flag.Bool("structs", true, "Whether or not to build structs")
	buildAPI := flag.Bool("api", true, "Whether or not to build the API")
	flag.Parse()
	files, err := ioutil.ReadDir(*swaggerDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, swaggerFile := range files {
		if !swaggerFile.IsDir() && strings.HasSuffix(swaggerFile.Name(), ".json") {
			apiBase := strings.TrimSuffix(swaggerFile.Name(), ".json")
			swaggerPath := strings.Join([]string{*swaggerDir, swaggerFile.Name()}, "/")
			swaggerString, err := ioutil.ReadFile(swaggerPath)
			if err != nil {
				continue
			}
			var s Swagger
			json.Unmarshal(swaggerString, &s)
			ParseModels(s.Models.(map[string]interface{}))
			BuildAPIs(apiBase, s)
		}
	}
	fmt.Println("package ari")
	if *buildAPI {
		fmt.Println(apiPreamble)
	}
	if *buildStructs {
		OutputStructs()
	}
	if *buildAPI {
		fmt.Print(clientAPIBuf.String())
	}
}
