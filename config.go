package ncp

import (
	"time"
)

type Config struct {
	SessionWindowSize            int32 // in bytes
	MTU                          int32 // in bytes
	InitialConnectionWindowSize  int32 // in packets
	MaxConnectionWindowSize      int32 // in packets
	MinConnectionWindowSize      int32 // in packets
	MaxAckSeqListSize            int32
	NonStream                    bool
	FlushInterval                time.Duration
	CloseDelay                   time.Duration
	InitialRetransmissionTimeout time.Duration
	MaxRetransmissionTimeout     time.Duration
	SendAckInterval              time.Duration
	CheckTimeoutInterval         time.Duration
	DialTimeout                  time.Duration
}
