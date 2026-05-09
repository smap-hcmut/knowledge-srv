// Package docs provides the Swagger registry required by gin-swagger.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
  "schemes": {{ marshal .Schemes }},
  "swagger": "2.0",
  "info": {
    "description": "{{escape .Description}}",
    "title": "{{.Title}}",
    "contact": {},
    "version": "{{.Version}}"
  },
  "host": "{{.Host}}",
  "basePath": "{{.BasePath}}",
  "paths": {}
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1",
	Host:             "",
	BasePath:         "/knowledge/api/v1/knowledge",
	Schemes:          []string{"https", "http"},
	Title:            "SMAP Knowledge Service API",
	Description:      "SMAP Knowledge Service API documentation.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
