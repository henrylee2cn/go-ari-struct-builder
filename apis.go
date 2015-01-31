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
	pathArgs := bytes.NewBufferString("")
	if o.ResponseClass != "void" {
		returnString = join("(", convertType(o.ResponseClass), ", error)")
	} else {
		returnString = "error"
	}
	go func(p chan string) {
		for s := range p {
			clientAPIBuf.WriteString(s)
		}
	}(p)
	funcName := join(Canonicalize(apiBase), Canonicalize(o.Nickname))
	funcArgs := bytes.NewBufferString("")
	setArgs := bytes.NewBufferString("")
	options := bytes.NewBufferString("for index, value := range options {\n")
	options.WriteString("switch index {\n")
	var argCount int = 0
	for _, param := range o.Parameters {
		if param.Required || param.ParamType == "path" {
			funcArgs.WriteString(join(Canonicalize(param.Name), " ", convertType(param.DataType), ","))
			if param.ParamType == "query" || param.ParamType == "body" {
				setArgs.WriteString(join("paramMap[\"", param.Name, "\"] = ", Canonicalize(param.Name), "\n"))
			} else if param.ParamType == "path" {
				replacedURL = strings.Replace(replacedURL, join("{", param.Name, "}"), "%s", 1)
				pathArgs.WriteString(join(", ", Canonicalize(param.Name)))
			}
		} else {
			options.WriteString(join("case ", strconv.Itoa(argCount),":\n"))
			options.WriteString("if len(value) > 0 {\n")
			options.WriteString(join("paramMap[\"", param.Name, "\"] = value\n"))
			options.WriteString("}\n")
			argCount++
		}
	}
	pathArgs.WriteString(")")
	options.WriteString("}\n}\n")
	setURL := join("url := fmt.Sprintf(\"", replacedURL, "\"", pathArgs.String(), "\n")
	funcArgs.WriteString(" options ...string")
	p <- join("func ", "(a *AppInstance) ", funcName, "(", funcArgs.String(),") ", returnString,"{\n")
	p <- join("paramMap := make(map[string]string)\n")
	p <- setArgs.String()
	p <- setURL
	p <- options.String()
	p <- "body := buildJSON(paramMap)\n"
	p <- join("result := a.processCommand(url, body, \"", o.HTTPMethod, "\")\n")
	p <- join("")

	if o.ResponseClass != "void" {
		p <- join("var r ", convertType(o.ResponseClass), "\n")
		p <- join("return json.Unmarshal(result.Body, &r), err\n")
	} else {
		p <- "return err\n"
	}
	p <- "}\n\n\n"
	p <- "\n"
}

func join(joinstrings ...string) string {
	return strings.Join(joinstrings, "")
}
