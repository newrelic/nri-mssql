package main

import (
	"testing"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name      string
		arg       *argumentList
		wantError bool
	}{
		{
			"No Errors",
			&argumentList{
				Username: "user",
				Hostname: "localhost",
				Port:     "90",
			},
			false,
		},
		{
			"No Username",
			&argumentList{
				Username: "",
				Hostname: "localhost",
				Port:     "90",
			},
			true,
		},
		{
			"No Hostname",
			&argumentList{
				Username: "user",
				Hostname: "",
				Port:     "90",
			},
			true,
		},
		{
			"No Port or Instance",
			&argumentList{
				Username: "user",
				Hostname: "localhost",
			},
			true,
		},
		{
			"Port and Instance",
			&argumentList{
				Username: "user",
				Hostname: "localhost",
				Port:     "90",
				Instance: "MSSQL",
			},
			true,
		},
		{
			"SSL and No Server Certificate",
			&argumentList{
				Username:               "user",
				Hostname:               "localhost",
				Port:                   "90",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "",
			},
			true,
		},
	}

	for _, tc := range testCases {
		err := tc.arg.Validate()
		if tc.wantError && err == nil {
			t.Errorf("Test Case %s Failed: Expected error", tc.name)
		} else if !tc.wantError && err != nil {
			t.Errorf("Test Case %s Failed: Unexpected error: %v", tc.name, err)
		}
	}
}
