package gogen

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	apiutil "github.com/zeromicro/go-zero/tools/goctl/api/util"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
)

const typesFile = "types"

//go:embed types.tpl
var typesTemplate string

// BuildTypes gen types to string
func BuildTypes(types []spec.Type, config *config.Config) (string, error) {
	var builder strings.Builder
	first := true
	for _, tp := range types {
		if first {
			first = false
		} else {
			builder.WriteString("\n\n")
		}
		if err := writeType(&builder, tp, config); err != nil {
			return "", apiutil.WrapErr(err, "Type "+tp.Name()+" generate error")
		}
	}

	return builder.String(), nil
}

func genTypes(dir string, cfg *config.Config, api *spec.ApiSpec) error {
	val, err := BuildTypes(api.Types, cfg)
	if err != nil {
		return err
	}

	typeFilename, err := format.FileNamingFormat(cfg.NamingFormat, typesFile)
	if err != nil {
		return err
	}

	typeFilename = typeFilename + ".go"
	filename := path.Join(dir, typesDir, typeFilename)
	os.Remove(filename)

	return genFile(fileGenConfig{
		dir:             dir,
		subdir:          typesDir,
		filename:        typeFilename,
		templateName:    "typesTemplate",
		category:        category,
		templateFile:    typesTemplateFile,
		builtinTemplate: typesTemplate,
		data: map[string]interface{}{
			"types":        val,
			"containsTime": false,
		},
	})
}

func writeType(writer io.Writer, tp spec.Type, config *config.Config) error {
	structType, ok := tp.(spec.DefineStruct)
	if !ok {
		return fmt.Errorf("unspport struct type: %s", tp.Name())
	}

	// write doc for swagger
	if config.AnnotateWithSwagger {
		inBodyTagCount := 0
		// if a response has more than one `in: body` tag, we should use `swagger:model` tag
		for _, member := range structType.Members {
			s := strings.Join(member.Docs, "")
			if s != "" && strings.Contains(s, "in: body") {
				inBodyTagCount++
			}
		}
		stringBuilder := &strings.Builder{}
		for _, v := range structType.Documents() {
			stringBuilder.WriteString(fmt.Sprintf("\t%s\n", v))
		}
		if strings.HasSuffix(tp.Name(), "Resp") {
			if stringBuilder.Len() > 0 {
				fmt.Fprintf(writer, stringBuilder.String())
			} else {
				fmt.Fprintf(writer, "\t// The response data of %s \n", strings.TrimSuffix(tp.Name(), "Resp"))
			}
			if inBodyTagCount > 1 {
				fmt.Fprintf(writer, "\t// swagger:model %s\n", tp.Name())
			} else {
				fmt.Fprintf(writer, "\t// swagger:response %s\n", tp.Name())
			}
		} else {
			if strings.HasSuffix(tp.Name(), "Req") {
				if stringBuilder.Len() > 0 {
					fmt.Fprintf(writer, stringBuilder.String())
				}
				if strings.HasSuffix(tp.Name(), "ParamReq") {
					// https://goswagger.io/use/spec/params.html
					// extract the operationId from the type name
					operationId := strings.TrimSuffix(tp.Name(), "ParamReq")
					fmt.Fprintf(writer, "\t// swagger:parameters %s %s\n ", tp.Name(), operationId)
				} else {
					fmt.Fprintf(writer, "\t// swagger:model %s\n", tp.Name())
				}
			} else {
				if stringBuilder.Len() > 0 {
					fmt.Fprintf(writer, stringBuilder.String())
				}
				fmt.Fprintf(writer, "\t// swagger:model %s\n", tp.Name())
			}
		}
	}

	fmt.Fprintf(writer, "type %s struct {\n", util.Title(tp.Name()))
	for _, member := range structType.Members {
		if member.IsInline {
			if _, err := fmt.Fprintf(writer, "%s\n", strings.Title(member.Type.Name())); err != nil {
				return err
			}

			continue
		}

		if err := writeProperty(writer, member.Name, member.Tag, member.GetComment(), member.Type, member.Docs, 1, config); err != nil {
			return err
		}
	}
	fmt.Fprintf(writer, "}")
	return nil
}
