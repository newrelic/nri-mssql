package metrics

import (
	"reflect"
	"strings"
	"testing"
)

func Test_QueryDefinition_GetQuery(t *testing.T) {
	expected := "select * from everywhere"
	def := QueryDefinition{
		query: expected,
	}

	if out := def.GetQuery(); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}

func Test_QueryDefinition_GetQuery_WithMod(t *testing.T) {
	expected := "select * from everywhere"
	def := QueryDefinition{
		query: "select %REPLACE% from everywhere",
	}

	modifier := func() QueryModifier {
		return func(query string) string {
			return strings.Replace(query, "%REPLACE%", "*", -1)
		}
	}

	if out := def.GetQuery(modifier()); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}

func Test_QueryDefinition_GetDataModels(t *testing.T) {
	input := []int{1, 2, 3, 4}

	control := make([]interface{}, len(input))

	for i, num := range input {
		control[i] = num
	}

	def := QueryDefinition{
		dataModels: &control,
	}

	out := def.GetDataModels()
	if reflect.DeepEqual(out, control) {
		t.Errorf("Expected %+v to not equal %+v", out, control)
	}
}
