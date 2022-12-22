// +build integration

package tests

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/stretchr/testify/assert"
)

const (
	containerName   = "nri-mssql"
	schema          = "mssql-schema.json"
	dbContainerName = "mssql"
	dbUsername      = "sa"
	dbPassword      = "secret123!"
)

func TestMain(m *testing.M) {
	flag.Parse()
	result := m.Run()
	os.Exit(result)
}

func waitForMSSQLIsUpAndRunning(maxTries int) bool {
	mssqlEnvVars := []string{
		"ACCEPT_EULA=Y",
		fmt.Sprintf("SA_PASSWORD=%s", dbPassword),
		"MSSQL_PID=Developer",
	}
	ports := []string{"1433:1433"}
	stdout, stderr, err := dockerComposeRunMode(mssqlEnvVars, ports, dbContainerName, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(stdout)
	fmt.Println(stderr)
	log.Info("try to establish de connection with the mssql database...")
	for ; maxTries > 0; maxTries-- {
		log.Info("try to establish de connection with the mssql database...")

		conn, err := connection.NewConnection(&args.ArgumentList{
			Username: dbUsername,
			Password: dbPassword,
			Hostname: "localhost",
			Timeout:  "2",
		})
		if err != nil {
			log.Warn(err.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		if conn != nil {
			conn.Close()
			log.Info("mssql is up & running!")
			return true
		}
	}
	return false
}

func TestSuccessConnection(t *testing.T) {
	if !waitForMSSQLIsUpAndRunning(20) {
		t.Fatal("tests cannot be executed")
	}
	envVars := []string{
		fmt.Sprintf("HOSTNAME=%s", dbContainerName),
		fmt.Sprintf("USERNAME=%s", dbUsername),
		fmt.Sprintf("PASSWORD=%s", dbPassword),
	}
	stdout, stderr, err := dockerComposeRun(envVars, containerName)
	t.Log(stdout)
	t.Log(stderr)
	assert.Nil(t, err)
	assert.NotEmpty(t, stdout)
	err = validateJSONSchema(schema, stdout)
	assert.Nil(t, err)
}
