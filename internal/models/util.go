package models

import (
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
)

func IDHash(id string) (string, error) {
	h := fnv.New64()
	_, err := h.Write([]byte(id))
	if err != nil {
		return "", err
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, h.Sum64())
	return hex.EncodeToString(b), nil
}
