package xendit

import "time"

type Config struct {
	APIKey         string
	APISecret      string
	RetryMax       int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}
