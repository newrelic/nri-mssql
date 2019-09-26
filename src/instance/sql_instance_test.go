package instance

import (
	"errors"
	"regexp"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/stretchr/testify/assert"
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
	rows := sqlmock.NewRows([]string{"instance_name"}).
		AddRow("testinstance")
	mock.ExpectQuery(regexp.QuoteMeta(instanceNameQuery)).WillReturnRows(rows)

	entity, err := CreateInstanceEntity(i, conn)
	assert.Nil(t, err)

	assert.Equal(t, "testhost", entity.Metadata.Name)
	assert.Equal(t, "ms-instance", entity.Metadata.Namespace)
	assert.Len(t, entity.Metadata.IDAttrs, 1)
}

func Test_createInstanceEntity_NullResponse(t *testing.T) {
	// create dummy integration
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := connection.CreateMockSQL(t)

	// set up sql mock
	rows := sqlmock.NewRows([]string{"instance_name"}).
		AddRow(nil)
	mock.ExpectQuery(regexp.QuoteMeta(instanceNameQuery)).WillReturnRows(rows)

	entity, err := CreateInstanceEntity(i, conn)
	assert.Nil(t, err)

	assert.Equal(t, "testhost", entity.Metadata.Name)
	assert.Equal(t, "ms-instance", entity.Metadata.Namespace)
	assert.Len(t, entity.Metadata.IDAttrs, 0)
}
