package gogen

import (
	_ "embed"
	"fmt"
	"path"
	"strings"

	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"github.com/zeromicro/go-zero/tools/goctl/pkg/golang"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
)

const defaultLogicPackage = "logic"

//go:embed handler.tpl
var handlerTemplate string

type handlerInfo struct {
	PkgName            string
	ImportPackages     string
	ImportHttpxPackage string
	HandlerDoc         string
	HandlerName        string
	RequestType        string
	LogicName          string
	LogicType          string
	Call               string
	HasResp            bool
	HasRequest         bool
}

func genHandler(dir, rootPkg string, cfg *config.Config, group spec.Group, route spec.Route) error {
	handler := getHandlerName(route)
	handlerPath := getHandlerFolderPath(group, route)
	pkgName := handlerPath[strings.LastIndex(handlerPath, "/")+1:]
	logicName := defaultLogicPackage
	if handlerPath != handlerDir {
		handler = strings.Title(handler)
		logicName = pkgName
	}
	parentPkg, err := golang.GetParentPackage(dir)
	if err != nil {
		return err
	}

	// write doc for swagger
	var handlerDoc *strings.Builder
	handlerDoc = &strings.Builder{}

	if cfg.AnnotateWithSwagger {
		var swaggerDescription string
		var swaggerSummary string
		var swaggerTags []string
		var swaggerOperationId string

		if value, ok := route.AtDoc.Properties["description"]; ok {
			swaggerDescription = value
		} else if value, ok := route.AtDoc.Properties["desc"]; ok {
			swaggerDescription = value
		} else {
			swaggerDescription = strings.TrimPrefix(route.HandlerDoc[0], "//")
		}

		if value, ok := route.AtDoc.Properties["summary"]; ok {
			swaggerSummary = value
		} else if value, ok := route.AtDoc.Properties["sum"]; ok {
			swaggerSummary = value
		} else {
			swaggerSummary = strings.TrimPrefix(route.HandlerDoc[0], "//")
		}

		if value, ok := route.AtDoc.Properties["tags"]; ok {
			swaggerTags = strings.Split(value, ",")
		} else if value, ok := route.AtDoc.Properties["tag"]; ok {
			swaggerTags = []string{value}
		} else {
			swaggerTags = []string{group.GetAnnotation("group")}
		}

		if value, ok := route.AtDoc.Properties["operationId"]; ok {
			swaggerOperationId = value
		} else {
			swaggerOperationId = strings.TrimSuffix(handler, "Handler")
		}

		swaggerTagValue := strings.Join(swaggerTags, " ")
		//swagger:route [method] [path pattern] [?tag1 tag2 tag3] [operation id]
		handlerDoc.WriteString(fmt.Sprintf("// swagger:route %s %s %s %s \n", route.Method, route.Path, swaggerTagValue, swaggerOperationId))
		handlerDoc.WriteString("//\n")
		handlerDoc.WriteString(fmt.Sprintf("// %s\n", swaggerDescription))
		handlerDoc.WriteString("//\n")
		handlerDoc.WriteString(fmt.Sprintf("// %s\n", swaggerSummary))
		handlerDoc.WriteString("//\n")

		// HasRequest
		if len(route.RequestTypeName()) > 0 && strings.ToUpper(route.Method) != "GET" {
			handlerDoc.WriteString(fmt.Sprintf(`// Parameters:
			//  + name: body
			//    require: true
			//    in: body
			//    type: %s
			//
    		//
			`, route.RequestTypeName()))
		}
		// HasResp
		if len(route.ResponseTypeName()) > 0 {
			handlerDoc.WriteString(fmt.Sprintf(`// Responses:
			//  200: %s
			//
			//`, route.ResponseTypeName()))
		}
	}

	return doGenToFile(dir, handler, cfg, group, route, handlerInfo{
		PkgName:        pkgName,
		ImportPackages: genHandlerImports(group, route, parentPkg),
		HandlerDoc:     handlerDoc.String(),
		HandlerName:    handler,
		RequestType:    util.Title(route.RequestTypeName()),
		LogicName:      logicName,
		LogicType:      strings.Title(getLogicName(route)),
		Call:           strings.Title(strings.TrimSuffix(handler, "Handler")),
		HasResp:        len(route.ResponseTypeName()) > 0,
		HasRequest:     len(route.RequestTypeName()) > 0,
	})
}

func doGenToFile(dir, handler string, cfg *config.Config, group spec.Group,
	route spec.Route, handleObj handlerInfo,
) error {
	filename, err := format.FileNamingFormat(cfg.NamingFormat, handler)
	if err != nil {
		return err
	}

	return genFile(fileGenConfig{
		dir:             dir,
		subdir:          getHandlerFolderPath(group, route),
		filename:        filename + ".go",
		templateName:    "handlerTemplate",
		category:        category,
		templateFile:    handlerTemplateFile,
		builtinTemplate: handlerTemplate,
		data:            handleObj,
	})
}

func genHandlers(dir, rootPkg string, cfg *config.Config, api *spec.ApiSpec) error {
	for _, group := range api.Service.Groups {
		for _, route := range group.Routes {
			if err := genHandler(dir, rootPkg, cfg, group, route); err != nil {
				return err
			}
		}
	}

	return nil
}

func genHandlerImports(group spec.Group, route spec.Route, parentPkg string) string {
	imports := []string{
		fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, getLogicFolderPath(group, route))),
		fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, contextDir)),
	}
	if len(route.RequestTypeName()) > 0 {
		imports = append(imports, fmt.Sprintf("\"%s\"\n", pathx.JoinPackages(parentPkg, typesDir)))
	}

	return strings.Join(imports, "\n\t")
}

func getHandlerBaseName(route spec.Route) (string, error) {
	handler := route.Handler
	handler = strings.TrimSpace(handler)
	handler = strings.TrimSuffix(handler, "handler")
	handler = strings.TrimSuffix(handler, "Handler")

	return handler, nil
}

func getHandlerFolderPath(group spec.Group, route spec.Route) string {
	folder := route.GetAnnotation(groupProperty)
	if len(folder) == 0 {
		folder = group.GetAnnotation(groupProperty)
		if len(folder) == 0 {
			return handlerDir
		}
	}

	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")

	return path.Join(handlerDir, folder)
}

func getHandlerName(route spec.Route) string {
	handler, err := getHandlerBaseName(route)
	if err != nil {
		panic(err)
	}

	return handler + "Handler"
}

func getLogicName(route spec.Route) string {
	handler, err := getHandlerBaseName(route)
	if err != nil {
		panic(err)
	}

	return handler + "Logic"
}
