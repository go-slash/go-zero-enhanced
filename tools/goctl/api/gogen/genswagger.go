package gogen

import (
	_ "embed"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"os"
	"path"
	"strconv"
	"strings"
)

//go:embed swagger.tpl
var swaggerTemplate string

//go:embed swagger-ui.tpl
var swaggerUITemplate string

func genSwagger(dir string, cfg *config.Config, spec *spec.ApiSpec) error {
	if !cfg.SwaggerAPIDocs {
		return nil
	}

	title, _ := strconv.Unquote(spec.Info.Properties["title"])
	version, _ := strconv.Unquote(spec.Info.Properties["version"])
	desc, _ := strconv.Unquote(spec.Info.Properties["desc"])
	contact, _ := strconv.Unquote(spec.Info.Properties["contact"])
	email, _ := strconv.Unquote(spec.Info.Properties["email"])
	basePath, _ := strconv.Unquote(spec.Info.Properties["basePath"])
	host, _ := strconv.Unquote(spec.Info.Properties["host"])

	swaggerObj := swaggerObject{
		Swagger:           "2.0",
		Schemes:           []string{"http", "https"},
		Consumes:          []string{"application/json"},
		Produces:          []string{"application/json"},
		Paths:             make(swaggerPathsObject),
		Definitions:       make(swaggerDefinitionsObject),
		StreamDefinitions: make(swaggerDefinitionsObject),
		Info: swaggerInfoObject{
			Title:       title,
			Version:     version,
			Description: desc,
			Contact: &swaggerContactObject{
				Name:  contact,
				Email: email,
			},
		},
		SecurityDefinitions: swaggerSecurityDefinitionsObject{
			"Bearer": swaggerSecuritySchemeObject{
				Name:        "Authorization",
				Description: "Enter JWT Bearer token ",
				Type:        "apiKey",
				In:          "header",
			},
		},
	}
	if len(host) > 0 {
		swaggerObj.Host = host
	}

	if len(basePath) > 0 {
		swaggerObj.BasePath = basePath
	}

	swaggerObj.Paths = renderServiceRoutes(spec.Service)

	swaggerObj.Definitions = renderTypeDefinition(spec.Types)

	swaggerDocFileName := path.Join(dir, swaggerDir, swaggerFileName)
	os.Remove(swaggerDocFileName)
	name := strings.ToLower(spec.Service.Name)

	if err := genFile(fileGenConfig{
		dir:             dir,
		subdir:          swaggerDir,
		filename:        swaggerHandlerFileName + ".go",
		templateName:    "swagger",
		category:        category,
		templateFile:    swaggerTemplateFile,
		builtinTemplate: swaggerTemplate,
		data: map[string]string{
			"serviceName": name,
		},
	}); err != nil {
		return err
	}

	if err := genPlainTextFile(fileGenConfig{
		dir:      dir,
		subdir:   swaggerDir,
		filename: swaggerUITemplateFileName,
		data:     swaggerUITemplate,
		json:     false,
	}); err != nil {
		return err
	}

	return genPlainTextFile(fileGenConfig{
		dir:      dir,
		subdir:   swaggerDir,
		filename: swaggerFileName,
		data:     swaggerObj,
		json:     true,
	})
}
