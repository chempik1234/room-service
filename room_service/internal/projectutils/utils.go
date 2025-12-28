package projectutils

import (
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"time"
)

// NowTimestamp returns current timestamp as int64
func NowTimestamp() int64 {
	return time.Now().Unix()
}

// GenerateRequestID generates requestID for logger.KeyForRequestID to store
func GenerateRequestID() string {
	return types.GenerateUUID().String()
}
