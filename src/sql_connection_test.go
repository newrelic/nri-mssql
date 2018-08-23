package main

import (
	"testing"
)

func Test_createConnectionURL(t *testing.T) {
	testCases := []struct {
		name string
		arg  *argumentList
		want string
	}{
		{
			"Port No SSL",
			&argumentList{
				Username:  "user",
				Password:  "pass",
				Hostname:  "localhost",
				EnableSSL: false,
				Port:      "1443",
				Timeout:   "30",
			},
			"sqlserver://user:pass@localhost:1443?dial+timeout=30",
		},
		{
			"Instance No SSL",
			&argumentList{
				Username:  "user",
				Password:  "pass",
				Hostname:  "localhost",
				EnableSSL: false,
				Instance:  "SQLExpress",
				Timeout:   "30",
			},
			"sqlserver://user:pass@localhost/SQLExpress?dial+timeout=30",
		},
		{
			"Instance SSL Trust",
			&argumentList{
				Username:               "user",
				Password:               "pass",
				Hostname:               "localhost",
				EnableSSL:              true,
				TrustServerCertificate: true,
				Instance:               "SQLExpress",
				Timeout:                "30",
			},
			"sqlserver://user:pass@localhost/SQLExpress?TrustServerCertificate=true&dial+timeout=30&encrypt=true",
		},
		{
			"Instance SSL Certificate",
			&argumentList{
				Username:               "user",
				Password:               "pass",
				Hostname:               "localhost",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "file.ca",
				Instance:               "SQLExpress",
				Timeout:                "30",
			},
			"sqlserver://user:pass@localhost/SQLExpress?TrustServerCertificate=false&certificate=file.ca&dial+timeout=30&encrypt=true",
		},
	}

	for _, tc := range testCases {
		args = *tc.arg
		if out := createConnectionURL(); out != tc.want {
			t.Errorf("Test Case %s Failed: Expected '%s' got '%s'", tc.name, tc.want, out)
		}
	}
}
