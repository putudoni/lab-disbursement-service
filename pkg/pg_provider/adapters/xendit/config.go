package xendit

import "time"

type Config struct {
	SecretKey      string
	RetryMax       int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}
