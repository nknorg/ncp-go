package ncp

import "github.com/imdario/mergo"

type Config struct {
	NonStream                    bool
	SessionWindowSize            int32 // in bytes
	MTU                          int32 // in bytes
	MinConnectionWindowSize      int32 // in packets
	MaxAckSeqListSize            int32
	FlushInterval                int32 // in millisecond
	Linger                       int32 // in millisecond
	InitialRetransmissionTimeout int32 // in millisecond
	MaxRetransmissionTimeout     int32 // in millisecond
	SendAckInterval              int32 // in millisecond
	CheckTimeoutInterval         int32 // in millisecond
	CheckBytesReadInterval       int32 // in millisecond
	SendBytesReadThreshold       int32 // in millisecond
	Verbose                      bool
}

var DefaultConfig = Config{
	NonStream:                    false,
	SessionWindowSize:            4 << 20,
	MTU:                          1024,
	MinConnectionWindowSize:      1,
	MaxAckSeqListSize:            32,
	FlushInterval:                10,
	Linger:                       1000,
	InitialRetransmissionTimeout: 5000,
	MaxRetransmissionTimeout:     10000,
	SendAckInterval:              50,
	CheckTimeoutInterval:         50,
	CheckBytesReadInterval:       100,
	SendBytesReadThreshold:       200,
	Verbose:                      false,
}

func MergeConfig(conf *Config) (*Config, error) {
	merged := DefaultConfig
	if conf != nil {
		err := mergo.Merge(&merged, conf, mergo.WithOverride)
		if err != nil {
			return nil, err
		}
	}
	return &merged, nil
}
