package main

import (
	"errors"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func Test_createInstanceEntity_QueryError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)

	mock.ExpectQuery(instanceNameQuery).WillReturnError(errors.New("error"))

	if _, err := createInstanceEntity(i, conn); err == nil {
		t.Error("Did not return expected error")
	}
}

func Test_createInstanceEntity_RowError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)

	rows := sqlmock.NewRows([]string{"instance_name"}).
		AddRow("my-instance").
		AddRow("other-instance")
	mock.ExpectQuery(instanceNameQuery).WillReturnRows(rows)

	if _, err := createInstanceEntity(i, conn); err == nil {
		t.Error("Did not return expected error")
	}
}
func Test_createInstanceEntity(t *testing.T) {
	// create dummy integration
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)

	// set up sql mock
	instanceName := "my-instance"
	rows := sqlmock.NewRows([]string{"instance_name"}).
		AddRow(instanceName)
	mock.ExpectQuery(instanceNameQuery).WillReturnRows(rows)

	entity, err := createInstanceEntity(i, conn)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	if entity.Metadata.Name != instanceName {
		t.Errorf("Expected entity name '%s' got '%s'", instanceName, entity.Metadata.Name)
	} else if entity.Metadata.Namespace != "instance" {
		t.Errorf("Expected entity namesapce 'instance' got '%s'", entity.Metadata.Namespace)
	}
}
