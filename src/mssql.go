package main

import (
	"os"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

const (
	integrationName    = "com.newrelic.nri-mssql"
	integrationVersion = "0.1.0"
)

var (
	args argumentList
)

func main() {
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	// Setup logging with verbose
	log.SetupLogging(args.Verbose)

	// Validate arguments
	if err := args.Validate(); err != nil {
		log.Error("Configuration error: %s", args)
		os.Exit(1)
	}

	// Create a new connection
	con, err := newConnection()
	if err != nil {
		log.Error("Error creating connection to SQL Server: %s", err.Error())
		os.Exit(1)
	}

	// Close connection when done
	defer con.Close()

	if err = i.Publish(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}