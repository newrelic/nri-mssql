package models

import (
	"fmt"
	"testing"
)

func runScanTests(t *testing.T, tests []struct {
	name    string
	input   interface{}
	want    string
	wantErr error
}, scanFunc func(interface{}) (string, error),
) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scanFunc(tt.input)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Scan() unexpected error = %v", err)
				}
				if got != tt.want {
					t.Errorf("Scan() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestHexString_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr error
	}{
		{
			name:    "valid byte slice",
			input:   []uint8{0x12, 0x34, 0xab, 0xcd},
			want:    "0x1234abcd",
			wantErr: nil,
		},
		{
			name:    "empty byte slice",
			input:   []uint8{},
			want:    "0x",
			wantErr: nil,
		},
		{
			name:    "invalid type",
			input:   "not a byte slice",
			want:    "",
			wantErr: fmt.Errorf("%w, got %T", ErrExpectedUint8Slice, "not a byte slice"),
		},
		{
			name:    "nil input",
			input:   nil,
			want:    "",
			wantErr: fmt.Errorf("%w, got %T", ErrExpectedUint8Slice, nil),
		},
	}

	runScanTests(t, tests, func(input interface{}) (string, error) {
		var hex HexString
		err := hex.Scan(input)
		return string(hex), err
	})
}

func TestVarBinary64_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr error
	}{
		{
			name:    "VarBinary64: valid byte slice",
			input:   []byte{0x12, 0x34, 0xab, 0xcd},
			want:    "0x1234abcd",
			wantErr: nil,
		},

		{
			name:    "input value is nil",
			input:   nil,
			want:    "",
			wantErr: fmt.Errorf("%w, got %T", ErrExpectedByteSlice, nil),
		},
	}

	runScanTests(t, tests, func(input interface{}) (string, error) {
		var vb VarBinary64
		err := vb.Scan(input)
		return string(vb), err
	})
}
