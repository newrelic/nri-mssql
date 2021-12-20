// Package args contains the argument list, defined as a struct, along with a method that validates passed-in args
package args

import (
	"errors"
	"github.com/newrelic/infra-integrations-sdk/log"
	"os"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
)

// ArgumentList struct that holds all MSSQL arguments
type ArgumentList struct {
	sdkArgs.DefaultArgumentList
	Username                     string `default:"" help:"The Microsoft SQL Server connection user name"`
	Password                     string `default:"" help:"The Microsoft SQL Server connection password"`
	Instance                     string `default:"" help:"The Microsoft SQL Server instance to connect to"`
	Database                     string `default:"" help:"The Microsoft SQL Server database to connect to"`
	Hostname                     string `default:"127.0.0.1" help:"The Microsoft SQL Server connection host name"`
	Port                         string `default:"" help:"The Microsoft SQL Server port to connect to. Only needed when instance not specified"`
	EnableSSL                    bool   `default:"false" help:"If true will use SSL encryption, false will not use encryption"`
	TrustServerCertificate       bool   `default:"false" help:"If true server certificate is not verified for SSL. If false certificate will be verified against supplied certificate"`
	CertificateLocation          string `default:"" help:"Certificate file to verify SSL encryption against"`
	EnableBufferMetrics          bool   `default:"true" help:"Enable collection of buffer space metrics."`
	EnableDatabaseReserveMetrics bool   `default:"true" help:"Enable collection of database reserve space metrics."`
	Timeout                      string `default:"30" help:"Timeout in seconds for a single SQL Query. Set 0 for no timeout"`
	CustomMetricsQuery           string `default:"" help:"A SQL query to collect custom metrics. Query results 'metric_name', 'metric_value', and 'metric_type' have special meanings"`
	CustomMetricsConfig          string `default:"" help:"YAML configuration with one or more SQL queries to collect custom metrics"`
	ShowVersion                  bool   `default:"false" help:"Print build information and exit"`
	ExtraConnectionURLArgs       string `default:"" help:"Appends additional parameters to connection url. Ex. 'applicationintent=readonly&foo=bar'"`
	QueryPlanConfig              string `default:"" help:"YAML configuration with one or more SQL queries that collects query plans"`
	LogApiEndpoint               string `default:"https://log-api.newrelic.com/log/v1" help:"Log API endpoint for Query Plans"`
	LicenseKey                   string `default:"" help:"New Relic License Key or Insights Insert Key"`
}

// Validate validates SQL specific arguments
func (al ArgumentList) Validate() error {

	if al.Hostname == "" {
		return errors.New("invalid configuration: must specify a hostname")
	}

	if al.Port != "" && al.Instance != "" {
		return errors.New("invalid configuration: specify either port or instance but not both")
	} else if al.Port == "" && al.Instance == "" {
		log.Info("Neither port nor instance were specified, using default port of 1433")
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

	if al.QueryPlanConfig != "" && al.LicenseKey == "" {
		return errors.New("LicenseKey required for Query Plan generation")
	}

	return nil
}
