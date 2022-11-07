package swagger

import (
	"bytes"
	_ "embed"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"text/template"
)

//go:embed swagger.json
var swaggerJson string

//go:embed swagger.tpl
var swaggerTemplateV2 string

type Opts func(*swaggerConfig)

// SwaggerOpts configures the Doc middlewares.
type swaggerConfig struct {
	// SpecURL the url to find the spec for
	SpecURL string
	// SwaggerHost for the js that generates the swagger ui site
	SwaggerHost string
    // SwaggerServerName for the js that generates the swagger ui site
	SwaggerServerName string
}

func Docs(basePath string, opts ...Opts) http.HandlerFunc {
	config := &swaggerConfig{
		SpecURL:     basePath + "/swagger.json",
		SwaggerHost: "https://cdn.bootcdn.net/ajax/libs/swagger-ui/4.15.2",
		SwaggerServerName: "{{.serviceName}}",
	}

	for _, opt := range opts {
		opt(config)
	}

	tmpl := template.Must(template.New("swaggerdoc").Parse(swaggerTemplateV2))
	buf := bytes.NewBuffer(nil)
	err := tmpl.Execute(buf, config)
	htmlText := buf.Bytes()

	return func(rw http.ResponseWriter, r *http.Request) {
		if err != nil {
			httpx.Error(rw, err)
			return
		}
		if r.URL.Path == basePath {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err = rw.Write(htmlText); err != nil {
				httpx.Error(rw, err)
				return
			}
			rw.WriteHeader(http.StatusOK)
		}
	}
}

func DocsJSON() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		if _, err := rw.Write([]byte(swaggerJson)); err != nil {
			http.NotFound(rw, r)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}
}
