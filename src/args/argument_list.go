// Package args contains the argument list, defined as a struct, along with a method that validates passed-in args
package args

import (
	"errors"
	"os"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/v3/args"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
)

// ArgumentList struct that holds all MSSQL arguments
type ArgumentList struct {
	sdkArgs.DefaultArgumentList
	Username                             string `default:"" help:"The Microsoft SQL Server connection user name"`
	Password                             string `default:"" help:"The Microsoft SQL Server connection password"`
	Instance                             string `default:"" help:"The Microsoft SQL Server instance to connect to"`
	Hostname                             string `default:"127.0.0.1" help:"The Microsoft SQL Server connection host name"`
	Port                                 string `default:"" help:"The Microsoft SQL Server port to connect to. Only needed when instance not specified"`
	EnableSSL                            bool   `default:"false" help:"If true will use SSL encryption, false will not use encryption"`
	TrustServerCertificate               bool   `default:"false" help:"If true server certificate is not verified for SSL. If false certificate will be verified against supplied certificate"`
	CertificateLocation                  string `default:"" help:"Certificate file to verify SSL encryption against"`
	EnableBufferMetrics                  bool   `default:"true" help:"Enable collection of buffer space metrics."`
	EnableDatabaseReserveMetrics         bool   `default:"true" help:"Enable collection of database reserve space metrics."`
	Timeout                              string `default:"30" help:"Timeout in seconds for a single SQL Query. Set 0 for no timeout"`
	CustomMetricsQuery                   string `default:"" help:"A SQL query to collect custom metrics. Query results 'metric_name', 'metric_value', and 'metric_type' have special meanings"`
	CustomMetricsConfig                  string `default:"" help:"YAML configuration with one or more SQL queries to collect custom metrics"`
	ShowVersion                          bool   `default:"false" help:"Print build information and exit"`
	ExtraConnectionURLArgs               string `default:"" help:"Appends additional parameters to connection url. Ex. 'applicationintent=readonly&foo=bar'"`
	EnableDiskMetricsInBytes             bool   `default:"true" help:"Enable collection of instance.diskInBytes."`
	EnableQueryMonitoring                bool   `default:"false" help:"Enable collection of detailed query performance metrics."`
	QueryMonitoringResponseTimeThreshold int    `default:"500" help:"Threshold in milliseconds for query response time. If response time exceeds this threshold, the query will be considered slow."`
	QueryMonitoringCountThreshold        int    `default:"20" help:"Maximum number of queries returned in query analysis results."`
	QueryMonitoringFetchInterval         int    `default:"15" help:"Interval in seconds for fetching grouped slow queries; Should always be same as mysql-config interval."`
}

// Validate validates SQL specific arguments
func (al ArgumentList) Validate() error {

	if al.Hostname == "" {
		return errors.New("invalid configuration: must specify a hostname")
	}

	if al.Port != "" && al.Instance != "" {
		return errors.New("invalid configuration: specify either port or instance but not both")
	} else if al.Port == "" && al.Instance == "" {
		log.Info("Both port and instance were not specified using default port of 1433")
		al.Port = "1433"
	}

	if al.EnableSSL && (!al.TrustServerCertificate && al.CertificateLocation == "") {
		return errors.New("invalid configuration: must specify a certificate file when using SSL and not trusting server certificate")
	}

	if len(al.CustomMetricsConfig) > 0 {
		if len(al.CustomMetricsQuery) > 0 {
			return errors.New("cannot specify options custom_metrics_query and custom_metrics_config")
		}
		if _, err := os.Stat(al.CustomMetricsConfig); err != nil {
			return errors.New("custom_metrics_config argument: " + err.Error())
		}
	}

	return nil
}
