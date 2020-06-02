package handler

import (
	"crypto/md5"
	"encoding/hex"
)

// createConfigHash hashes compiled config (JSON) using md5
func createConfigHash(config []byte) string {
	h := md5.New()
	h.Write(config)
	return hex.EncodeToString(h.Sum(nil))
}
