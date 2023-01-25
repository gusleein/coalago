package coalago

import (
	"time"
)

var (
	timeWait        = time.Second
	maxSendAttempts = 6
	sumTimeAttempts = timeWait*time.Duration(maxSendAttempts) + 100
)

const (
	SESSIONS_POOL_EXPIRATION = time.Second * 60 * 2
	MAX_PAYLOAD_SIZE         = 1024
	DEFAULT_WINDOW_SIZE      = 300
	MIN_WiNDOW_SIZE          = 50
	MAX_WINDOW_SIZE          = 1500
	MTU                      = 1500
)

var NumberConnections = 1024
