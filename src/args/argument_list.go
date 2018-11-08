// Package args contains the argument list, defined as a struct, along with a method that validates passed-in args
package args

import (
	"errors"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// ArgumentList struct that holds all MSSQL arguments
type ArgumentList struct {
	sdkArgs.DefaultArgumentList
	Username               string `default:"" help:"The Microsoft SQL Server connection user name"`
	Password               string `default:"" help:"The Microsoft SQL Server connection password"`
	Instance               string `default:"" help:"The Microsoft SQL Server instance to connect to"`
	Hostname               string `default:"127.0.0.1" help:"The Microsoft SQL Server connection host name"`
	Port                   string `default:"" help:"The Microsoft SQL Server port to connect to. Only needed when instance not specified"`
	EnableSSL              bool   `default:"false" help:"If true will use SSL encryption, false will not use encryption"`
	TrustServerCertificate bool   `default:"false" help:"If true server certificate is not verified for SSL. If false certificate will be verified against supplied certificate"`
	CertificateLocation    string `default:"" help:"Certificate file to verify SSL encryption against"`
	Timeout                string `default:"30" help:"Timeout in seconds for a single SQL Query. Set 0 for no timeout"`
}

// Validate validates SQL specific arguments
func (al ArgumentList) Validate() error {
	if al.Username == "" {
		return errors.New("invalid configuration: must specify a username")
	}

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

	return nil
}
