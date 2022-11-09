package gogen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"net/http"
	"reflect"
	"strings"
)

const (
	defaultOption   = "default"
	optionalOption  = "optional"
	omitemptyOption = "omitempty"
	optionsOption   = "options"
	exampleOption   = "example"
	optionSeparator = "|"
	equalToken      = "="
)

func renderServiceRoutes(service spec.Service) (pathsObject swaggerPathsObject) {
	pathsObject = make(swaggerPathsObject)
	groups := service.Groups
	for _, group := range groups {
		for _, targetRoute := range group.Routes {
			path := group.GetAnnotation("prefix") + targetRoute.Path
			if path[0] != '/' {
				path = "/" + path
			}
			parameters := swaggerParametersObject{}
			// extract path parameters
			if strings.Contains(path, ":") {
				pathParts := strings.Split(path, "/")
				for i := range pathParts {
					part := pathParts[i]
					if strings.Contains(part, ":") {
						key := strings.TrimPrefix(pathParts[i], ":")
						// path placeholder replace from :key to {key}
						path = strings.Replace(path, fmt.Sprintf(":%s", key), fmt.Sprintf("{%s}", key), 1)
						pObject := swaggerParameterObject{
							Name:     key,
							In:       "path",
							Required: true,
							Type:     "string",
						}
						// extract path parameter description if exists in atDoc property
						if desc, ok := targetRoute.AtDoc.Properties[key]; ok {
							pObject.Description = strings.Trim(desc, "\"")
						}
						parameters = append(parameters, pObject)
					}
				}
			}

			// check if the route has a request body with DefineStruct
			if defineStruct, ok := targetRoute.RequestType.(spec.DefineStruct); ok {
				//todo: extract parameters from the request headers and add them to the parameters list

				// extract from the query parameters if it is a struct
				if strings.ToUpper(targetRoute.Method) == http.MethodGet {
					for _, member := range defineStruct.Members {
						if member.IsPathMember() {
							continue
						}
						if embedStruct, isEmbed := member.Type.(spec.DefineStruct); isEmbed {
							for _, m := range embedStruct.Members {
								parameters = append(parameters, renderQueryStruct(m))
							}
							continue
						}
						parameters = append(parameters, renderQueryStruct(member))
					}
				} else if len(defineStruct.GetBodyMembers()) > 0 {
					// skip generating query struct if not body parameters exists

					// extract from the request body if it is a struct
					reqRef := fmt.Sprintf("#/definitions/%s", targetRoute.RequestType.Name())

					if len(targetRoute.RequestType.Name()) > 0 {
						schema := swaggerSchemaObject{
							schemaCore: schemaCore{
								Ref: reqRef,
							},
						}

						parameter := swaggerParameterObject{
							Name:     "body",
							In:       "body",
							Required: true,
							Schema:   &schema,
						}

						doc := strings.Join(targetRoute.RequestType.Documents(), ",")
						doc = strings.Replace(doc, "//", "", -1)

						if doc != "" {
							parameter.Description = doc
						}

						parameters = append(parameters, parameter)
					}
				}
			}

			// if path not exists, create path item object
			if _, ok := pathsObject[path]; !ok {
				pathsObject[path] = swaggerPathItemObject{}
			}

			pathItemObject := pathsObject[path]

			desc := "A successful response."
			respRef := ""

			if targetRoute.ResponseType != nil && len(targetRoute.ResponseType.Name()) > 0 {
				respRef = fmt.Sprintf("#/definitions/%s", targetRoute.ResponseType.Name())
			}

			tags := getRouteTag(service.Name, targetRoute, group)

			operationObject := &swaggerOperationObject{
				OperationID: targetRoute.Handler,
				Tags:        []string{tags},
				Parameters:  parameters,
				Responses: swaggerResponsesObject{
					"200": swaggerResponseObject{
						Description: desc,
						Schema: swaggerSchemaObject{
							schemaCore: schemaCore{
								Ref: respRef,
							},
						},
					},
				},
				Description: getRouteDescription(targetRoute),
				Summary:     getRouteSummary(targetRoute),
			}

			if group.Annotation.Properties["jwt"] != "" {
				operationObject.Security = &[]swaggerSecurityRequirementObject{{"Bearer": []string{}}}
			}

			switch strings.ToUpper(targetRoute.Method) {
			case http.MethodGet:
				pathItemObject.Get = operationObject
			case http.MethodPost:
				pathItemObject.Post = operationObject
			case http.MethodDelete:
				pathItemObject.Delete = operationObject
			case http.MethodPut:
				pathItemObject.Put = operationObject
			case http.MethodPatch:
				pathItemObject.Patch = operationObject
			}

			pathsObject[path] = pathItemObject
		}
	}

	return
}

func renderQueryStruct(member spec.Member) swaggerParameterObject {
	tempKind := swaggerMapTypes[strings.Replace(member.Type.Name(), "[]", "", -1)]

	pType, format, ok := convertGoTypeToSchemaType(tempKind, member.Type.Name())

	if !ok {
		pType = tempKind.String()
		format = "UNKNOWN"
	}

	pObject := swaggerParameterObject{In: "query", Type: pType, Format: format}

	pObject.Required = !member.IsOptionalInForm()

	if value, err := member.GetPropertyDefaultValue(); err == nil {
		pObject.Default = value
	}

	name, err := member.GetPropertyName()

	if err == nil {
		pObject.Name = name
	}

	if len(member.Comment) > 0 {
		pObject.Description = strings.TrimLeft(member.Comment, "//")
	}

	return pObject
}

func renderTypeDefinition(p []spec.Type) swaggerDefinitionsObject {
	d := make(map[string]swaggerSchemaObject)

	for _, pType := range p {
		schema := swaggerSchemaObject{
			schemaCore: schemaCore{
				Type: "object",
			},
		}
		defineStruct, ok := pType.(spec.DefineStruct)

		if !ok {
			continue // skip if not a DefineStruct
		}

		schema.Title = defineStruct.Name()

		for _, member := range defineStruct.Members {
			if hasPathParameters(member) {
				continue
			}

			kv := keyVal{Value: schemaOfField(member)}
			kv.Key = member.Name

			if tag, err := member.GetPropertyName(); err == nil {
				kv.Key = tag
			}

			// if member is embed struct, add all members to the schema
			if kv.Key == "" {
				memberStruct, _ := member.Type.(spec.DefineStruct)

				for _, m := range memberStruct.Members {

					if strings.Contains(m.Tag, "header") ||
						strings.Contains(m.Tag, "form") ||
						strings.Contains(m.Tag, "path") {
						continue
					}

					mkv := keyVal{
						Value: schemaOfField(m),
						Key:   m.Name,
					}

					if tag, err := m.GetPropertyName(); err == nil {
						mkv.Key = tag
					}

					if schema.Properties == nil {
						schema.Properties = &swaggerSchemaObjectProperties{}
					}

					*schema.Properties = append(*schema.Properties, mkv)
				}
				continue
			}

			if schema.Properties == nil {
				schema.Properties = &swaggerSchemaObjectProperties{}
			}

			*schema.Properties = append(*schema.Properties, kv)

			for _, tag := range member.Tags() {
				if len(tag.Options) == 0 {
					if !contains(schema.Required, tag.Name) && tag.Name != "required" {
						schema.Required = append(schema.Required, tag.Name)
					}
					continue
				}

				required := true
				for _, option := range tag.Options {
					if strings.HasPrefix(option, optionalOption) || strings.HasPrefix(option, omitemptyOption) {
						required = false
					}
				}

				if required && !contains(schema.Required, tag.Name) {
					schema.Required = append(schema.Required, tag.Name)
				}
			}

		}

		d[pType.Name()] = schema
	}

	return d
}

func getRouteTag(serviceName string, route spec.Route, group spec.Group) string {
	var tag = serviceName
	if val, ok := route.AtDoc.Properties["tag"]; ok {
		tag = val
	} else if val, ok := route.AtDoc.Properties["tags"]; ok {
		tag = val
	} else if val, ok := group.Annotation.Properties["group"]; ok {
		tag = val
	}
	return strings.Trim(tag, "\"")
}

func getRouteSummary(route spec.Route) string {
	var summary = ""

	if val, ok := route.AtDoc.Properties["summary"]; ok {
		summary = val
	} else if val, ok := route.AtDoc.Properties["description"]; ok {
		summary = val
	}

	return strings.Trim(summary, "\"")
}

func getRouteDescription(route spec.Route) string {
	var desc = ""
	if val, ok := route.AtDoc.Properties["description"]; ok {
		desc = val
	} else if val, ok := route.AtDoc.Properties["desc"]; ok {
		desc = val
	}
	return strings.Trim(desc, "\"")
}

func hasPathParameters(member spec.Member) bool {
	for _, tag := range member.Tags() {
		if tag.Key == "path" {
			return true
		}
	}

	return false
}

func schemaOfField(member spec.Member) swaggerSchemaObject {
	ret := swaggerSchemaObject{}

	var core schemaCore

	kind := swaggerMapTypes[member.Type.Name()]
	var props *swaggerSchemaObjectProperties

	comment := member.GetComment()
	comment = strings.Replace(comment, "//", "", -1)

	switch ft := kind; ft {
	case reflect.Invalid: //[]Struct 也有可能是 Struct
		// []Struct
		// map[ArrayType:map[Star:map[StringExpr:UserSearchReq] StringExpr:*UserSearchReq] StringExpr:[]*UserSearchReq]
		refTypeName := strings.Replace(member.Type.Name(), "[", "", 1)
		refTypeName = strings.Replace(refTypeName, "]", "", 1)
		refTypeName = strings.Replace(refTypeName, "*", "", 1)
		refTypeName = strings.Replace(refTypeName, "{", "", 1)
		refTypeName = strings.Replace(refTypeName, "}", "", 1)
		// interface

		if refTypeName == "interface" {
			core = schemaCore{Type: "object"}
		} else if refTypeName == "mapstringstring" {
			core = schemaCore{Type: "object"}
		} else if strings.HasPrefix(refTypeName, "[]") {
			core = schemaCore{Type: "array"}

			tempKind := swaggerMapTypes[strings.Replace(refTypeName, "[]", "", -1)]
			ftype, format, ok := convertGoTypeToSchemaType(tempKind, refTypeName)
			if ok {
				core.Items = &swaggerItemsObject{Type: ftype, Format: format}
			} else {
				core.Items = &swaggerItemsObject{Type: ft.String(), Format: "UNKNOWN"}
			}

		} else {
			core = schemaCore{
				Ref: "#/definitions/" + refTypeName,
			}
		}
	case reflect.Slice:
		tempKind := swaggerMapTypes[strings.Replace(member.Type.Name(), "[]", "", -1)]
		ftype, format, ok := convertGoTypeToSchemaType(tempKind, member.Type.Name())

		if ok {
			core = schemaCore{Type: ftype, Format: format}
		} else {
			core = schemaCore{Type: ft.String(), Format: "UNKNOWN"}
		}
	default:
		ftype, format, ok := convertGoTypeToSchemaType(ft, member.Type.Name())
		if ok {
			core = schemaCore{Type: ftype, Format: format}
		} else {
			core = schemaCore{Type: ft.String(), Format: "UNKNOWN"}
		}
	}

	switch ft := kind; ft {
	case reflect.Slice:
		ret = swaggerSchemaObject{
			schemaCore: schemaCore{
				Type:  "array",
				Items: (*swaggerItemsObject)(&core),
			},
		}
	case reflect.Invalid:
		// 判断是否数组
		if strings.HasPrefix(member.Type.Name(), "[]") {
			ret = swaggerSchemaObject{
				schemaCore: schemaCore{
					Type:  "array",
					Items: (*swaggerItemsObject)(&core),
				},
			}
		} else {
			ret = swaggerSchemaObject{
				schemaCore: core,
				Properties: props,
			}
		}
		if strings.HasPrefix(member.Type.Name(), "map") {
			fmt.Println("暂不支持map类型")
		}
	default:
		ret = swaggerSchemaObject{
			schemaCore: core,
			Properties: props,
		}
	}
	ret.Description = comment

	for _, tag := range member.Tags() {
		if len(tag.Options) == 0 {
			continue
		}
		for _, option := range tag.Options {
			switch {
			case strings.HasPrefix(option, defaultOption):
				parts := strings.Split(option, equalToken)
				if len(parts) == 2 {
					ret.Default = parts[1]
				}
			case strings.HasPrefix(option, optionsOption):
				parts := strings.SplitN(option, equalToken, 2)
				if len(parts) == 2 {
					ret.Enum = strings.Split(parts[1], optionSeparator)
				}
			case strings.HasPrefix(option, exampleOption):
				parts := strings.Split(option, equalToken)
				if len(parts) == 2 {
					ret.Example = parts[1]
				}
			}
		}
	}

	return ret
}

// https://swagger.io/specification/ Data Types
func convertGoTypeToSchemaType(kind reflect.Kind, t string) (pType, format string, ok bool) {
	switch kind {
	case reflect.Int:
		return "integer", "int32", true
	case reflect.Uint:
		return "integer", "uint32", true
	case reflect.Int8:
		return "integer", "int8", true
	case reflect.Uint8:
		return "integer", "uint8", true
	case reflect.Int16:
		return "integer", "int16", true
	case reflect.Uint16:
		return "integer", "uin16", true
	case reflect.Int64:
		return "integer", "int64", true
	case reflect.Uint64:
		return "integer", "uint64", true
	case reflect.Bool:
		return "boolean", "boolean", true
	case reflect.String:
		return "string", "", true
	case reflect.Float32:
		return "number", "float", true
	case reflect.Float64:
		return "number", "double", true
	case reflect.Slice:
		return strings.Replace(t, "[]", "", -1), "", true
	default:
		return "", "", false
	}
}

var swaggerMapTypes = map[string]reflect.Kind{
	"string":   reflect.String,
	"*string":  reflect.String,
	"int":      reflect.Int,
	"*int":     reflect.Int,
	"uint":     reflect.Uint,
	"*uint":    reflect.Uint,
	"int8":     reflect.Int8,
	"*int8":    reflect.Int8,
	"uint8":    reflect.Uint8,
	"*uint8":   reflect.Uint8,
	"int16":    reflect.Int16,
	"*int16":   reflect.Int16,
	"uint16":   reflect.Uint16,
	"*uint16":  reflect.Uint16,
	"int32":    reflect.Int,
	"*int32":   reflect.Int,
	"uint32":   reflect.Int,
	"*uint32":  reflect.Int,
	"uint64":   reflect.Int64,
	"*uint64":  reflect.Int64,
	"int64":    reflect.Int64,
	"*int64":   reflect.Int64,
	"[]string": reflect.Slice,
	"[]int":    reflect.Slice,
	"[]int64":  reflect.Slice,
	"[]int32":  reflect.Slice,
	"[]uint32": reflect.Slice,
	"[]uint64": reflect.Slice,
	"bool":     reflect.Bool,
	"*bool":    reflect.Bool,
	"struct":   reflect.Struct,
	"*struct":  reflect.Struct,
	"float32":  reflect.Float32,
	"*float32": reflect.Float32,
	"float64":  reflect.Float64,
	"*float64": reflect.Float64,
}

// http://swagger.io/specification/#infoObject
type swaggerInfoObject struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
	Version        string `json:"version"`

	Contact *swaggerContactObject `json:"contact,omitempty"`
	License *swaggerLicenseObject `json:"license,omitempty"`
}

// http://swagger.io/specification/#contactObject
type swaggerContactObject struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// http://swagger.io/specification/#licenseObject
type swaggerLicenseObject struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// http://swagger.io/specification/#externalDocumentationObject
type swaggerExternalDocumentationObject struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// http://swagger.io/specification/#swaggerObject
type swaggerObject struct {
	Swagger             string                              `json:"swagger"`
	Info                swaggerInfoObject                   `json:"info"`
	Host                string                              `json:"host,omitempty"`
	BasePath            string                              `json:"basePath,omitempty"`
	Schemes             []string                            `json:"schemes"`
	Consumes            []string                            `json:"consumes"`
	Produces            []string                            `json:"produces"`
	Paths               swaggerPathsObject                  `json:"paths"`
	Definitions         swaggerDefinitionsObject            `json:"definitions"`
	StreamDefinitions   swaggerDefinitionsObject            `json:"x-stream-definitions,omitempty"`
	SecurityDefinitions swaggerSecurityDefinitionsObject    `json:"securityDefinitions,omitempty"`
	Security            []swaggerSecurityRequirementObject  `json:"security,omitempty"`
	ExternalDocs        *swaggerExternalDocumentationObject `json:"externalDocs,omitempty"`
}

// http://swagger.io/specification/#securityDefinitionsObject
type swaggerSecurityDefinitionsObject map[string]swaggerSecuritySchemeObject

// http://swagger.io/specification/#securitySchemeObject
type swaggerSecuritySchemeObject struct {
	Type             string              `json:"type"`
	Description      string              `json:"description,omitempty"`
	Name             string              `json:"name,omitempty"`
	In               string              `json:"in,omitempty"`
	Flow             string              `json:"flow,omitempty"`
	AuthorizationURL string              `json:"authorizationUrl,omitempty"`
	TokenURL         string              `json:"tokenUrl,omitempty"`
	Scopes           swaggerScopesObject `json:"scopes,omitempty"`
}

// http://swagger.io/specification/#scopesObject
type swaggerScopesObject map[string]string

// http://swagger.io/specification/#securityRequirementObject
type swaggerSecurityRequirementObject map[string][]string

// http://swagger.io/specification/#pathsObject
type swaggerPathsObject map[string]swaggerPathItemObject

// http://swagger.io/specification/#pathItemObject
type swaggerPathItemObject struct {
	Get    *swaggerOperationObject `json:"get,omitempty"`
	Delete *swaggerOperationObject `json:"delete,omitempty"`
	Post   *swaggerOperationObject `json:"post,omitempty"`
	Put    *swaggerOperationObject `json:"put,omitempty"`
	Patch  *swaggerOperationObject `json:"patch,omitempty"`
}

// http://swagger.io/specification/#operationObject
type swaggerOperationObject struct {
	Summary     string                  `json:"summary,omitempty"`
	Description string                  `json:"description,omitempty"`
	OperationID string                  `json:"operationId"`
	Responses   swaggerResponsesObject  `json:"responses"`
	Parameters  swaggerParametersObject `json:"parameters,omitempty"`
	RequestBody *struct {
		Content swaggerContentObject `json:"content,omitempty"`
	} `json:"requestBody,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Deprecated bool     `json:"deprecated,omitempty"`

	Security     *[]swaggerSecurityRequirementObject `json:"security,omitempty"`
	ExternalDocs *swaggerExternalDocumentationObject `json:"externalDocs,omitempty"`
}

type (
	swaggerParametersObject []swaggerParameterObject
	swaggerContentObject    map[string]swaggerParametersObject
)

// http://swagger.io/specification/#parameterObject
type swaggerParameterObject struct {
	Name             string              `json:"name"`
	Description      string              `json:"description,omitempty"`
	In               string              `json:"in,omitempty"`
	Required         bool                `json:"required"`
	Type             string              `json:"type,omitempty"`
	Format           string              `json:"format,omitempty"`
	Items            *swaggerItemsObject `json:"items,omitempty"`
	Enum             []string            `json:"enum,omitempty"`
	CollectionFormat string              `json:"collectionFormat,omitempty"`
	Default          string              `json:"default,omitempty"`
	MinItems         *int                `json:"minItems,omitempty"`
	Example          string              `json:"example,omitempty"`

	// Or you can explicitly refer to another type. If this is defined all
	// other fields should be empty
	Schema *swaggerSchemaObject `json:"schema,omitempty"`
}

// core part of schema, which is common to itemsObject and schemaObject.
// http://swagger.io/specification/#itemsObject
type schemaCore struct {
	Type    string `json:"type,omitempty"`
	Format  string `json:"format,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	Example string `json:"example,omitempty"`

	Items *swaggerItemsObject `json:"items,omitempty"`
	// If the item is an enumeration include a list of all the *NAMES* of the
	// enum values.  I'm not sure how well this will work but assuming all enums
	// start from 0 index it will be great. I don't think that is a good assumption.
	Enum    []string `json:"enum,omitempty"`
	Default string   `json:"default,omitempty"`
}

type swaggerItemsObject schemaCore

// http://swagger.io/specification/#responsesObject
type swaggerResponsesObject map[string]swaggerResponseObject

// http://swagger.io/specification/#responseObject
type swaggerResponseObject struct {
	Description string              `json:"description"`
	Schema      swaggerSchemaObject `json:"schema"`
}

type keyVal struct {
	Key   string
	Value interface{}
}

type swaggerSchemaObjectProperties []keyVal

func (op swaggerSchemaObjectProperties) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, kv := range op {
		if i != 0 {
			buf.WriteString(",")
		}
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")
		val, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

// http://swagger.io/specification/#schemaObject
type swaggerSchemaObject struct {
	schemaCore
	// Properties can be recursively defined
	Properties           *swaggerSchemaObjectProperties `json:"properties,omitempty"`
	AdditionalProperties *swaggerSchemaObject           `json:"additionalProperties,omitempty"`

	Description string `json:"description,omitempty"`
	Title       string `json:"title,omitempty"`

	ExternalDocs *swaggerExternalDocumentationObject `json:"externalDocs,omitempty"`

	ReadOnly         bool     `json:"readOnly,omitempty"`
	MultipleOf       float64  `json:"multipleOf,omitempty"`
	Maximum          float64  `json:"maximum,omitempty"`
	ExclusiveMaximum bool     `json:"exclusiveMaximum,omitempty"`
	Minimum          float64  `json:"minimum,omitempty"`
	ExclusiveMinimum bool     `json:"exclusiveMinimum,omitempty"`
	MaxLength        uint64   `json:"maxLength,omitempty"`
	MinLength        uint64   `json:"minLength,omitempty"`
	Pattern          string   `json:"pattern,omitempty"`
	MaxItems         uint64   `json:"maxItems,omitempty"`
	MinItems         uint64   `json:"minItems,omitempty"`
	UniqueItems      bool     `json:"uniqueItems,omitempty"`
	MaxProperties    uint64   `json:"maxProperties,omitempty"`
	MinProperties    uint64   `json:"minProperties,omitempty"`
	Required         []string `json:"required,omitempty"`
}

// http://swagger.io/specification/#definitionsObject
type swaggerDefinitionsObject map[string]swaggerSchemaObject

// Internal type to store used references.
type refMap map[string]struct{}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
