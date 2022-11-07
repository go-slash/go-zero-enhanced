package main

import (
	"flag"
	"fmt"

	{{.importPackages}}
)

var configFile = flag.String("f", "etc/{{.serviceName}}.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	{{if .enableSwagger}}
	if c.Mode == service.DevMode {
		server.AddRoutes([]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/swagger",
				Handler: swagger.Docs("/swagger"),
			},
			{
				Method:  http.MethodGet,
				Path:    "/swagger/swagger.json",
				Handler: swagger.DocsJSON(),
			},
		})
		fmt.Printf("swagger is running at http://%s:%d/swagger\n", c.Host, c.Port)
	}{{end}}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
