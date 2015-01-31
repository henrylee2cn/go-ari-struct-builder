package main

import (
	"bytes"
	"strings"
	"strconv"
)

func BuildAPIs(apiBase string, s Swagger) {
	for _, api := range s.APIs {
		curPath := api.Path
		for _, operation := range api.Operations {
			BuildClientFunc(apiBase, curPath, operation)
		}
	}
}

func BuildClientFunc(apiBase string, curPath string, o Operation) {
	var returnString string
	p := make(chan string)
	replacedURL := curPath
	var hasOptions bool = false
	var hasErrorResponses bool = false
	optionBlock := bytes.NewBufferString("")
	pathArgs := bytes.NewBufferString("")
	errorBlock := bytes.NewBufferString("")
	if o.ResponseClass != "void" {
		returnString = join("(*", convertType(o.ResponseClass), ", error)")
	} else {
		returnString = "error"
	}
	go func(p chan string) {
		for s := range p {
			clientAPIBuf.WriteString(s)
		}
	}(p)
	for _, param := range o.Parameters {
		if !(param.Required || param.ParamType == "path") {
			hasOptions = true
		}
	}
	funcName := join(Canonicalize(apiBase), Canonicalize(o.Nickname))
	funcArgs := bytes.NewBufferString("")
	setArgs := bytes.NewBufferString("")
	if hasOptions {
		optionBlock.WriteString("for index, value := range options {\n")
		optionBlock.WriteString("switch index {\n")
	}
	var argCount int = 0
	for _, param := range o.Parameters {
		if param.Required || param.ParamType == "path" {
			pName := Canonicalize(param.Name)
			if pName == "Variable" {
				pName = "Var"
			}
			funcArgs.WriteString(join(pName, " ", convertType(param.DataType), ","))
			if param.ParamType == "query" || param.ParamType == "body" {
				setArgs.WriteString(join("paramMap[\"", param.Name, "\"] = ", pName, "\n"))
			} else if param.ParamType == "path" {
				replacedURL = strings.Replace(replacedURL, join("{", param.Name, "}"), "%s", 1)
				pathArgs.WriteString(join(", ", pName))
			}
		} else {
			optionBlock.WriteString(join("case ", strconv.Itoa(argCount),":\n"))
			optionBlock.WriteString("if len(value) > 0 {\n")
			optionBlock.WriteString(join("paramMap[\"", param.Name, "\"] = value\n"))
			optionBlock.WriteString("}\n")
			argCount++
		}
	}
	pathArgs.WriteString(")")
	if hasOptions {
		optionBlock.WriteString("}\n}\n")
		funcArgs.WriteString(" options ...string")
	} else {
		a := funcArgs.String()
		funcArgs.Reset()
		funcArgs.WriteString(strings.TrimRight(a, ","))
	}
	if o.ErrorResponses != nil {
		hasErrorResponses = true
		errorBlock.WriteString("switch result.StatusCode {\n")
	}
	for _, errorResponse := range o.ErrorResponses {
		errorBlock.WriteString(join ("case ", strconv.Itoa(errorResponse.Code), ":\n"))
		errorBlock.WriteString(join("err = errors.New(\"", errorResponse.Reason, "\")\n"))
	}
	if hasErrorResponses {
		errorBlock.WriteString("default:\n")
		errorBlock.WriteString("err = nil\n")
		errorBlock.WriteString("}\n")
	}
	setURL := join("url := fmt.Sprintf(\"", replacedURL, "\"", pathArgs.String(), "\n")
	p <- join("func ", "(a *AppInstance) ", funcName, "(", funcArgs.String(),") ", returnString,"{\n")
	p <- "var err error\n"
	p <- join("paramMap := make(map[string]string)\n")
	p <- setArgs.String()
	p <- setURL
	p <- optionBlock.String()
	p <- "body := buildJSON(paramMap)\n"
	p <- join("result := a.processCommand(url, body, \"", o.HTTPMethod, "\")\n")
	p <- join("")
	if hasErrorResponses {
		p <- errorBlock.String()
	} else {
		p <- "err = nil\n"
	}
	if o.ResponseClass != "void" {
		p <- join("var r *", convertType(o.ResponseClass), "\n")
		p <- "json.Unmarshal([]byte(result.ResponseBody), r)\n"
		p <- "return r, err\n"
	} else {
		p <- "return err\n"
	}
	p <- "}\n\n\n"
	p <- "\n"
}

func join(joinstrings ...string) string {
	return strings.Join(joinstrings, "")
}
