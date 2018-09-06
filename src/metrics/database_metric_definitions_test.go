package metrics

import (
	"fmt"
	"testing"
)

func Test_dbNameReplace(t *testing.T) {

	dbName, format := "master", "use %s select * from %s"
	query := fmt.Sprintf(format, databasePlaceHolder, databasePlaceHolder)
	expected := fmt.Sprintf(format, dbName, dbName)

	modifier := dbNameReplace(dbName)
	if out := modifier(query); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}
