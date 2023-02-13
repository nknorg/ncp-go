package ncp

var (
	ErrSessionClosed         = NewGenericError("session closed", false, false)
	ErrSessionEstablished    = NewGenericError("session is already established", false, false)
	ErrSessionNotEstablished = NewGenericError("session not established yet", false, true)
	ErrReadDeadlineExceeded  = NewGenericError("read deadline exceeded", true, true)
	ErrWriteDeadlineExceeded = NewGenericError("write deadline exceeded", true, true)
	ErrBufferSizeTooSmall    = NewGenericError("read buffer size is less than data length in non-stream mode", false, true)
	ErrDataSizeTooLarge      = NewGenericError("data size is greater than session mtu", false, true)
	ErrInvalidPacket         = NewGenericError("invalid packet", false, true)
	ErrRecvWindowFull        = NewGenericError("receive window full", false, true)
	ErrNotHandshake          = NewGenericError("first packet is not handshake packet", false, true)
	ErrConnClosed            = NewGenericError("connection closed", false, false)
	ErrConnNotFound          = NewGenericError("connection not found", false, false)
)

type GenericError struct {
	err       string
	timeout   bool
	temporary bool
}

func NewGenericError(err string, timeout, temporary bool) *GenericError {
	return &GenericError{
		err:       err,
		timeout:   timeout,
		temporary: temporary,
	}
}

func (e GenericError) Error() string   { return e.err }
func (e GenericError) Timeout() bool   { return e.timeout }
func (e GenericError) Temporary() bool { return e.temporary }
