package models

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// these are custom type for handling SQL Server varbinary(64) fields.
type HexString string
type VarBinary64 string

// Define static errors
var (
	ErrExpectedUint8Slice = errors.New("HexString: expected []uint8")
	ErrExpectedByteSlice  = errors.New("VarBinary64: expected []byte")
)

// Scan implements the sql.Scanner interface for HexString
func (h *HexString) Scan(value interface{}) error {
	bytes, ok := value.([]uint8)
	if !ok {
		return fmt.Errorf("%w, got %T", ErrExpectedUint8Slice, value)
	}

	hexString := "0x" + hex.EncodeToString(bytes)
	*h = HexString(hexString)
	return nil
}

// Implement the sql.Scanner interface
func (vb *VarBinary64) Scan(value interface{}) error {
	bytes, ok := value.([]byte) // SQL drivers often use []byte for varbinary
	if !ok {
		return fmt.Errorf("%w, got %T", ErrExpectedByteSlice, value)
	}

	// Convert the []byte to a hex string with "0x" prefix
	*vb = VarBinary64("0x" + hex.EncodeToString(bytes))
	return nil
}
