package ncp

var (
	ErrDialTimeout           = GenericError{err: "dial timeout", timeout: true, temporary: true}
	ErrSessionClosed         = GenericError{err: "session closed", timeout: false, temporary: false}
	ErrSessionEstablished    = GenericError{err: "session is already established", timeout: false, temporary: false}
	ErrSessionNotEstablished = GenericError{err: "session not established yet", timeout: false, temporary: true}
	ErrReadDeadlineExceeded  = GenericError{err: "read deadline exceeded", timeout: true, temporary: true}
	ErrWriteDeadlineExceeded = GenericError{err: "write deadline exceeded", timeout: true, temporary: true}
	ErrBufferSizeTooSmall    = GenericError{err: "read buffer size is less than data length in non-session mode", timeout: false, temporary: true}
	ErrDataSizeTooLarge      = GenericError{err: "data size is greater than session mtu in non-session mode", timeout: false, temporary: true}
)

type GenericError struct {
	err       string
	timeout   bool
	temporary bool
}

func (e GenericError) Error() string   { return e.err }
func (e GenericError) Timeout() bool   { return e.timeout }
func (e GenericError) Temporary() bool { return e.temporary }
