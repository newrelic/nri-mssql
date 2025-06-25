package args

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			false,
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

func TestGetMaxConcurrentWorkers(t *testing.T) {
	testCases := []struct {
		name                 string
		maxConcurrentWorkers int
		expected             int
	}{
		{
			name:                 "Default value when not set (zero value)",
			maxConcurrentWorkers: 0,
			expected:             10,
		},
		{
			name:                 "Custom value",
			maxConcurrentWorkers: 15,
			expected:             15,
		},
		{
			name:                 "Negative value returns default",
			maxConcurrentWorkers: -5,
			expected:             10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := ArgumentList{
				Hostname:             "localhost",
				MaxConcurrentWorkers: tc.maxConcurrentWorkers,
			}

			result := args.GetMaxConcurrentWorkers()
			assert.Equal(t, tc.expected, result)
		})
	}
}
