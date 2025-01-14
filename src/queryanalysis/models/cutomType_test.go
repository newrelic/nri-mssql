package models

import (
	"fmt"
	"testing"
)

func TestHexString_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    HexString
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h HexString
			err := h.Scan(tt.input)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("HexString.Scan() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("HexString.Scan() unexpected error = %v", err)
				}
				if h != tt.want {
					t.Errorf("HexString.Scan() = %v, want %v", h, tt.want)
				}
			}
		})
	}
}

func TestVarBinary64_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    VarBinary64
		wantErr error
	}{
		{
			name:    "valid byte slice",
			input:   []byte{0x12, 0x34, 0xab, 0xcd},
			want:    "0x1234abcd",
			wantErr: nil,
		},
		{
			name:    "empty byte slice",
			input:   []byte{},
			want:    "0x",
			wantErr: nil,
		},
		{
			name:    "invalid type",
			input:   "not a byte slice",
			want:    "",
			wantErr: fmt.Errorf("%w, got %T", ErrExpectedByteSlice, "not a byte slice"),
		},
		{
			name:    "nil input",
			input:   nil,
			want:    "",
			wantErr: fmt.Errorf("%w, got %T", ErrExpectedByteSlice, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vb VarBinary64
			err := vb.Scan(tt.input)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("VarBinary64.Scan() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("VarBinary64.Scan() unexpected error = %v", err)
				}
				if vb != tt.want {
					t.Errorf("VarBinary64.Scan() = %v, want %v", vb, tt.want)
				}
			}
		})
	}
}
