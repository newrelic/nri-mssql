package args

import (
	"testing"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name      string
		arg       *ArgumentList
		wantError bool
	}{
		{
			"No Errors",
			&ArgumentList{
				Username: "user",
				Hostname: "localhost",
				Port:     "90",
			},
			false,
		},
		{
			"No Username",
			&ArgumentList{
				Username: "",
				Hostname: "localhost",
				Port:     "90",
			},
			true,
		},
		{
			"No Hostname",
			&ArgumentList{
				Username: "user",
				Hostname: "",
				Port:     "90",
			},
			true,
		},
		{
			"No Port or Instance",
			&ArgumentList{
				Username: "user",
				Hostname: "localhost",
			},
			false,
		},
		{
			"Port and Instance",
			&ArgumentList{
				Username: "user",
				Hostname: "localhost",
				Port:     "90",
				Instance: "MSSQL",
			},
			true,
		},
		{
			"SSL and No Server Certificate",
			&ArgumentList{
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
