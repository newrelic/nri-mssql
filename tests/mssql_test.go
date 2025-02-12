//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/tests/simulation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

var (
	oldestSupportedPerf = "mssql-perf-oldest"
	latestSupportedPerf = "mssql-perf-latest"
	unsupportedPerf     = "mssql-noext"
	// This is a known issue where latest versions of Ubuntu get a fatal error with MSSQL 2017
	// https://github.com/actions/runner-images/issues/10649#issuecomment-2380651901
	// When testing on x86 macs or debian linux both versions can be enabled
	// perfContainers       = []string{oldestSupportedPerf, latestSupportedPerf}
	perfContainers       = []string{latestSupportedPerf}
	nonPerfContainers    = []string{unsupportedPerf}
	integrationContainer = "nri_mssql"
	querySimCount        = 3

	defaultBinPath = "/nri-mssql"
	defaultUser    = "sa"         // Default MSSQL admin user
	defaultPass    = "secret123!" // Common MSSQL container password
	defaultPort    = 1433         // Default MSSQL port
	defaultDB      = "master"

	// cli flags
	container  = flag.String("container", integrationContainer, "container where the integration is installed")
	binaryPath = flag.String("bin", defaultBinPath, "Integration binary path")
	user       = flag.String("user", defaultUser, "MSSQL user name")
	psw        = flag.String("psw", defaultPass, "MSSQL user password")
	port       = flag.Int("port", defaultPort, "MSSQL port")
	database   = flag.String("database", defaultDB, "MSSQL database")

	allSampleTypes = []string{
		"MssqlInstanceSample",
		"MSSQLQueryExecutionPlans",
		"MSSQLTopSlowQueries",
		"MSSQLWaitTimeAnalysis",
		"MSSQLBlockingSessionQueries",
	}
	instanceSampleOnly = []string{
		"MssqlInstanceSample",
	}
)

func TestMain(m *testing.M) {
	flag.Parse()
	result := m.Run()
	os.Exit(result)
}

func TestIntegrationSupportedDatabase(t *testing.T) {
	tests := []struct {
		name                string
		containers          []string
		args                []string
		expectedSampleTypes []string
	}{
		{
			name:                "Perf metrics on supported database with perf enabled",
			containers:          perfContainers,
			args:                []string{`-enable_query_monitoring=true`},
			expectedSampleTypes: allSampleTypes,
		},
		{
			name:                "Perf metrics on supported database with perf enabled and custom parameters",
			containers:          perfContainers,
			args:                []string{`-enable_query_monitoring=true`, `-query_monitoring_response_time_threshold=2000`},
			expectedSampleTypes: allSampleTypes,
		},
		{
			name:                "Perf metrics on supported database with perf enabled and more custom parameters",
			containers:          perfContainers,
			args:                []string{`-enable_query_monitoring=true`, `-query_monitoring_response_time_threshold=10000`, `query_monitoring_fetch_interval=5`, `-query_monitoring_count_threshold=10`},
			expectedSampleTypes: []string{"MssqlInstanceSample", "MSSQLWaitTimeAnalysis", "MSSQLBlockingSessionQueries"},
		},
		{
			name:                "Perf metrics on supported database with perf disabled",
			containers:          perfContainers,
			args:                []string{`-enable_query_monitoring=false`},
			expectedSampleTypes: instanceSampleOnly,
		},
		{
			name:                "Perf metrics on supported database with perf disabled and more custom parameters",
			containers:          perfContainers,
			args:                []string{`-enable_query_monitoring=false`, `-query_monitoring_response_time_threshold=500`, `query_monitoring_fetch_interval=5`, `-query_monitoring_count_threshold=10`},
			expectedSampleTypes: instanceSampleOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, container := range tt.containers {
				t.Run(container, func(t *testing.T) {

					runQueries(t, container)

					stdout := runIntegration(t, container, tt.args...)

					samples := strings.Split(stdout, "\n")
					foundSampleTypes := make(map[string]bool)

					for _, sample := range samples {
						sample = strings.TrimSpace(sample)
						if sample == "" {
							continue
						}

						sampleType := getSampleType(sample, allSampleTypes)
						require.NotEmpty(t, sampleType, "No sample type found in JSON output: %s", sample)
						require.Contains(t, tt.expectedSampleTypes, sampleType, "Found unexpected sample type %q", sampleType)
						foundSampleTypes[sampleType] = true

						validateSample(t, sample, sampleType)
					}
					samplesFound := getFoundSampleTypes(foundSampleTypes)
					require.ElementsMatch(t, tt.expectedSampleTypes, samplesFound, "Not all expected sample types where found expected %v, found %v", tt.expectedSampleTypes, samplesFound)

				})
			}
		})
	}
}

func runQueries(t *testing.T, container string) {
	t.Helper()
	containerPort, containerDB := getPortAndDbForContainer(container)
	for i := 0; i < querySimCount; i++ {
		err := simulation.SimulateDBQueries(t, containerPort, containerDB, *user, *psw)
		require.NoError(t, err, "Failed to simulate database queries")
	}
}

func validateSample(t *testing.T, sample string, sampleType string) {
	t.Helper()
	// Validate JSON format
	var j map[string]interface{}
	err := json.Unmarshal([]byte(sample), &j)
	require.NoError(t, err, "Got an invalid JSON as output: %s", sample)

	// Validate schema
	t.Run(fmt.Sprintf("Validating JSON schema for sample: %s", sampleType), func(t *testing.T) {
		schemaFile := getSchemaFileName(sampleType)
		require.NotEmpty(t, schemaFile, "Schema file not found for sample type: %s", sampleType)

		err = validateJSONSchema(schemaFile, sample)
		assert.NoError(t, err, "Sample failed schema validation for type: %s", sampleType)
	})
}

func TestIntegrationUnsupportedDatabase(t *testing.T) {
	tests := []struct {
		name       string
		containers []string
		args       []string
	}{
		{
			name:       "Performance metrics collection with unsupported database with perf enabled",
			containers: nonPerfContainers,
			args:       []string{`-enable_query_monitoring=true`},
		},
		{
			name:       "Performance metrics collection with unsupported database with perf disabled",
			containers: nonPerfContainers,
			args:       []string{`-enable_query_monitoring=false`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, container := range tt.containers {
				t.Run(container, func(t *testing.T) {
					stdout := runIntegration(t, container, tt.args...)

					// Validate JSON format
					var j map[string]interface{}
					err := json.Unmarshal([]byte(stdout), &j)
					assert.NoError(t, err, "Integration Output Is An Invalid JSON")

					// Verify it's a MssqlInstanceSample
					assert.Contains(t, stdout, "MssqlInstanceSample",
						"Integration output does not contain MssqlInstanceSample")

					// Validate against schema
					err = validateJSONSchema("mssql-schema.json", stdout)
					assert.NoError(t, err, "Output failed schema validation")
				})
			}
		})
	}
}

// ---------------- HELPER FUNCTIONS ----------------

func getPortAndDbForContainer(container string) (int, string) {
	switch container {
	case latestSupportedPerf:
		return 1433, "AdventureWorks2022"
	case oldestSupportedPerf:
		return 2433, "AdventureWorks2017"
	case unsupportedPerf:
		return 3433, "master"
	default:
		return 1433, "master"
	}
}

func getSchemaFileName(sampleType string) string {
	schemaMap := map[string]string{
		"MssqlInstanceSample":         "mssql-schema.json",
		"MSSQLQueryExecutionPlans":    "execution-plan-schema.json",
		"MSSQLTopSlowQueries":         "slow-queries-schema.json",
		"MSSQLWaitTimeAnalysis":       "wait-events-schema.json",
		"MSSQLBlockingSessionQueries": "blocking-sessions-schema.json",
	}
	return schemaMap[sampleType]
}

func getFoundSampleTypes(foundSampleTypes map[string]bool) []string {
	foundSampleTypesKeys := make([]string, 0)
	for key, _ := range foundSampleTypes {
		foundSampleTypesKeys = append(foundSampleTypesKeys, key)
	}
	return foundSampleTypesKeys
}

func getSampleType(sample string, sampleTypes []string) string {
	for _, sType := range sampleTypes {
		if strings.Contains(sample, sType) {
			return sType
		}
	}
	return ""
}

func ExecInContainer(container string, command []string, envVars ...string) (string, string, error) {
	// No changes needed for this function
	cmdLine := make([]string, 0, 3+len(command))
	cmdLine = append(cmdLine, "exec", "-i")

	for _, envVar := range envVars {
		cmdLine = append(cmdLine, "-e", envVar)
	}

	cmdLine = append(cmdLine, container)
	cmdLine = append(cmdLine, command...)

	log.Debug("executing: docker %s", strings.Join(cmdLine, " "))

	cmd := exec.Command("docker", cmdLine...)

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdout := outbuf.String()
	stderr := errbuf.String()

	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

func runIntegration(t *testing.T, targetContainer string, integration_args ...string) string {
	t.Helper()

	command := make([]string, 0)
	command = append(command, *binaryPath)

	if user != nil {
		command = append(command, "-username", *user)
	}
	if psw != nil {
		command = append(command, "-password", *psw)
	}

	// Use default MSSQL port
	command = append(command, "-port", "1433")

	if targetContainer != "" {
		command = append(command, "-hostname", targetContainer)
	}

	for _, arg := range integration_args {
		command = append(command, arg)
	}

	stdout, stderr, err := ExecInContainer(*container, command)
	if stderr != "" {
		log.Debug("Integration command Standard Error: ", stderr)
	}
	fmt.Println(stderr)
	require.NoError(t, err)

	return stdout
}

func validateJSONSchema(fileName string, input string) error {
	pwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	schemaURI := fmt.Sprintf("file://%s", filepath.Join(pwd, "testdata", fileName))
	log.Info("loading schema from %s", schemaURI)
	schemaLoader := gojsonschema.NewReferenceLoader(schemaURI)
	documentLoader := gojsonschema.NewStringLoader(input)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("Error loading JSON schema, error: %v", err)
	}

	if result.Valid() {
		return nil
	}
	fmt.Printf("Errors for JSON schema: '%s'\n", schemaURI)
	for _, desc := range result.Errors() {
		fmt.Printf("\t- %s\n", desc)
	}
	fmt.Printf("\n")
	return fmt.Errorf("The output of the integration doesn't have expected JSON format")
}
