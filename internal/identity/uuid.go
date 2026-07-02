package identity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

func NewUUID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		fallback := fmt.Sprintf("%d:%d", time.Now().UTC().UnixNano(), os.Getpid())
		sum := fallbackBytes(fallback)
		copy(raw[:], sum[:])
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	)
}

func fallbackBytes(seed string) [16]byte {
	var out [16]byte
	for i := range out {
		out[i] = seed[i%len(seed)]
	}
	return out
}
