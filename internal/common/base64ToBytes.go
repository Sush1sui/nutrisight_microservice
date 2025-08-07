package common

import (
	"encoding/base64"
)

func Base64ToBytes(s string) ([]byte, error) {
    return base64.StdEncoding.DecodeString(s)
}