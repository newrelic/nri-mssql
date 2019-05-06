package instance

import (
	"errors"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func Test_createInstanceEntity_QueryError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := connection.CreateMockSQL(t)

	mock.ExpectQuery(instanceNameQuery).WillReturnError(errors.New("error"))

	if _, err := CreateInstanceEntity(i, conn); err == nil {
		t.Error("Did not return expected error")
	}
}

func Test_createInstanceEntity_RowError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := connection.CreateMockSQL(t)

	rows := sqlmock.NewRows([]string{"instance_name"}).
		AddRow("my-instance").
		AddRow("other-instance")
	mock.ExpectQuery(instanceNameQuery).WillReturnRows(rows)

	if _, err := CreateInstanceEntity(i, conn); err == nil {
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

	conn, mock := connection.CreateMockSQL(t)

	// set up sql mock
	instanceName := "testhost"
	rows := sqlmock.NewRows([]string{"instance_name"}).
		AddRow(instanceName)
	mock.ExpectQuery(instanceNameQuery).WillReturnRows(rows)

	entity, err := CreateInstanceEntity(i, conn)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	if entity.Metadata.Name != instanceName {
		t.Errorf("Expected entity name '%s' got '%s'", instanceName, entity.Metadata.Name)
	} else if entity.Metadata.Namespace != "ms-instance" {
		t.Errorf("Expected entity namespace 'instance' got '%s'", entity.Metadata.Namespace)
	}
}
