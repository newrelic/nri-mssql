package metrics

import (
	"fmt"
	"testing"
)

func Test_dbNameReplace(t *testing.T) {
	// baseline: normal name is inserted verbatim (no ] to escape)
	dbName, format := "master", "use [%s] select * from [%s]"
	query := fmt.Sprintf("use [%s] select * from [%s]", databasePlaceholder, databasePlaceholder)
	expected := fmt.Sprintf(format, dbName, dbName)

	modifier := dbNameReplace(dbName)
	if out := modifier(query); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}

func Test_dbNameReplace_escapesClosingBracket(t *testing.T) {
	// A ] in the name must be doubled so it cannot close the bracket-quoted identifier early.
	modifier := dbNameReplace("test]db")
	query := fmt.Sprintf("USE [%s]", databasePlaceholder)
	out := modifier(query)
	// ] → ]] so the identifier becomes [test]]db] which SQL Server reads as test]db
	if out != "USE [test]]db]" {
		t.Errorf("Expected 'USE [test]]db]' got '%s'", out)
	}
}

func Test_dbNameReplace_neutralisesInjection(t *testing.T) {
	// An attacker-controlled name containing SQL metacharacters must not produce
	// a second executable statement when embedded inside [bracket-quoted] USE.
	// The payload has no ] so no escaping is needed; the whole string is absorbed
	// inside [brackets] as a single identifier — the ; and -- are not executable.
	injected := `master";INSERT INTO evil SELECT secret FROM victim;--`
	modifier := dbNameReplace(injected)
	query := fmt.Sprintf("USE [%s]", databasePlaceholder)
	out := modifier(query)

	// The entire name must be wrapped inside [brackets] as one identifier.
	expected := fmt.Sprintf("USE [%s]", injected)
	if out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}
