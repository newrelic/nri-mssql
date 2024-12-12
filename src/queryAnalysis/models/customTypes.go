package models

import (
	"encoding/hex"
	"fmt"
)

type HexString string

// Scan implements the sql.Scanner interface for HexString
func (h *HexString) Scan(value interface{}) error {
	bytes, ok := value.([]uint8)
	if !ok {
		return fmt.Errorf("HexString: expected []uint8, got %T", value)
	}

	hexString := "0x" + hex.EncodeToString(bytes)
	*h = HexString(hexString)
	return nil
}
