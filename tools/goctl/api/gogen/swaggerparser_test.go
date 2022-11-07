package gogen

import (
	_ "embed"
	"github.com/zeromicro/go-zero/tools/goctl/api/parser"
	"testing"

	"github.com/stretchr/testify/assert"
	"os"
)

var (
	//go:embed testdata/swagger/basic.api
	testSwaggerBasicTemplate string
)

func TestBasicParser(t *testing.T) {
	filename := "greet.api"
	err := os.WriteFile(filename, []byte(testSwaggerBasicTemplate), os.ModePerm)
	assert.Nil(t, err)
	defer os.Remove(filename)

	spec, err := parser.Parse(filename)
	assert.Nil(t, err)

	object := renderServiceRoutes(spec.Service)

	assert.NotNil(t, object)

	//  tags
	assert.Equal(t, object["/greet/from/{name}"].Get.Tags[0], "greet")
	assert.Equal(t, object["/greet/from/{name}"].Get.Description, "Greet someone description")
	assert.Equal(t, object["/greet/from/{name}"].Get.Summary, "Greet someone summary")

	assert.Equal(t, object["/greet2/from/{name}"].Get.Tags[0], "example")

	defineObject := renderTypeDefinition(spec.Types)
	assert.NotNil(t, defineObject)

}
