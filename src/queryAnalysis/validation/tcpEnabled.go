package validation

import (
	"bytes"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"os/exec"
	"strings"
)

func checkTcpEnabled(sqlConnection *connection.SQLConnection) (bool, error) {
	var isTCPEnabled bool
	query := `
		SELECT CASE
			WHEN SERVERPROPERTY('IsClustered') = 1 OR SERVERPROPERTY('IsHadrEnabled') = 1
			THEN 1
			ELSE 0
		END AS is_tcp_enabled
	`
	err := sqlConnection.Connection.Get(&isTCPEnabled, query)
	if err != nil {
		return false, err
	}

	if !isTCPEnabled {
		log.Error("You have not enabled TCP for remote access. Please refer to the documentation: https://learn.microsoft.com/en-us/sql/database-engine/configure-windows/configure-a-server-to-listen-on-a-specific-tcp-port?view=sql-server-ver16")
	}

	return isTCPEnabled, nil
}

// CheckFirewallSettings checks if a firewall rule exists for allowing TCP traffic on port 1433
func CheckFirewallSettings() error {
	psScript := `
  $ruleName = "SQL Server TCP 1433"
  $rule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
  if ($rule -eq $null) {
   Write-Output "Rule not found"
  } else {
   Write-Output "Rule exists"
  }
 `

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return err
	}

	result := strings.TrimSpace(out.String())
	if result == "Rule not found" {
		log.Error("Firewall rule to allow TCP traffic on port 1433 not found. Please ensure that your firewall allows traffic on port 1433.")
	} else {
		log.Info("Firewall rule exists to allow TCP traffic on port 1433.")
	}

	return nil
}
