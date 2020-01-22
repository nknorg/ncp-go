package ncp

type Config struct {
	NonStream                    bool
	SessionWindowSize            int32 // in bytes
	MTU                          int32 // in bytes
	InitialConnectionWindowSize  int32 // in packets
	MaxConnectionWindowSize      int32 // in packets
	MinConnectionWindowSize      int32 // in packets
	MaxAckSeqListSize            int32
	FlushInterval                int32 // in millisecond
	Linger                       int32 // in millisecond
	InitialRetransmissionTimeout int32 // in millisecond
	MaxRetransmissionTimeout     int32 // in millisecond
	SendAckInterval              int32 // in millisecond
	CheckTimeoutInterval         int32 // in millisecond
	DialTimeout                  int32 // in millisecond
}
