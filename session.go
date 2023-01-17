package ncp

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nknorg/ncp-go/pb"
)

const (
	MinSequenceID = 1
)

type Session struct {
	config           *Config
	localAddr        net.Addr
	remoteAddr       net.Addr
	localClientIDs   []string
	remoteClientIDs  []string
	sendWith         SendWithFunc
	sendWindowSize   uint32
	recvWindowSize   uint32
	sendMtu          uint32
	recvMtu          uint32
	connections      map[string]*Connection
	onAccept         chan struct{}
	sendChan         chan uint32
	resendChan       chan uint32
	sendWindowUpdate chan struct{}
	recvDataUpdate   chan struct{}
	context          context.Context
	cancel           context.CancelFunc
	readContext      context.Context
	readCancel       context.CancelFunc
	writeContext     context.Context
	writeCancel      context.CancelFunc
	readLock         sync.Mutex
	writeLock        sync.Mutex

	acceptLock sync.Mutex
	isAccepted bool

	sync.RWMutex
	isEstablished       bool
	isClosed            bool
	sendBuffer          []byte
	sendWindowStartSeq  uint32
	sendWindowEndSeq    uint32
	sendWindowData      map[uint32][]byte
	recvWindowStartSeq  uint32
	recvWindowUsed      uint32
	recvWindowData      map[uint32][]byte
	bytesWrite          uint64
	bytesRead           uint64
	bytesReadSentTime   time.Time
	bytesReadUpdateTime time.Time
	remoteBytesRead     uint64

	sendWindowPacketCount float64 // Equal to sendWindowsSize / sendMtu
}

type SendWithFunc func(localClientID, remoteClientID string, buf []byte, writeTimeout time.Duration) error

func NewSession(localAddr, remoteAddr net.Addr, localClientIDs, remoteClientIDs []string, sendWith SendWithFunc, config *Config) (*Session, error) {
	config, err := MergeConfig(config)
	if err != nil {
		return nil, err
	}

	session := &Session{
		config:                config,
		localAddr:             localAddr,
		remoteAddr:            remoteAddr,
		localClientIDs:        localClientIDs,
		remoteClientIDs:       remoteClientIDs,
		sendWith:              sendWith,
		sendWindowSize:        uint32(config.SessionWindowSize),
		recvWindowSize:        uint32(config.SessionWindowSize),
		sendMtu:               uint32(config.MTU),
		recvMtu:               uint32(config.MTU),
		sendWindowStartSeq:    MinSequenceID,
		sendWindowEndSeq:      MinSequenceID,
		recvWindowStartSeq:    MinSequenceID,
		recvWindowUsed:        0,
		bytesWrite:            0,
		bytesRead:             0,
		bytesReadSentTime:     time.Now(),
		bytesReadUpdateTime:   time.Now(),
		remoteBytesRead:       0,
		onAccept:              make(chan struct{}, 1),
		sendWindowPacketCount: float64(config.SessionWindowSize) / float64(config.MTU),
	}

	session.context, session.cancel = context.WithCancel(context.Background())
	err = session.SetDeadline(zeroTime)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (session *Session) IsStream() bool {
	return !session.config.NonStream
}

func (session *Session) IsEstablished() bool {
	session.RLock()
	defer session.RUnlock()
	return session.isEstablished
}

func (session *Session) IsClosed() bool {
	session.RLock()
	defer session.RUnlock()
	return session.isClosed
}

func (session *Session) GetBytesRead() uint64 {
	session.RLock()
	defer session.RUnlock()
	return session.bytesRead
}

func (session *Session) updateBytesReadSentTime() {
	session.Lock()
	session.bytesReadSentTime = time.Now()
	session.Unlock()
}

func (session *Session) SendWindowUsed() uint32 {
	session.RLock()
	defer session.RUnlock()
	if session.bytesWrite > session.remoteBytesRead {
		return uint32(session.bytesWrite - session.remoteBytesRead)
	}
	return 0
}

func (session *Session) RecvWindowUsed() uint32 {
	session.RLock()
	defer session.RUnlock()
	return session.recvWindowUsed
}

func (session *Session) GetDataToSend(sequenceID uint32) []byte {
	session.RLock()
	defer session.RUnlock()
	return session.sendWindowData[sequenceID]
}

func (session *Session) GetConnWindowSize() uint32 {
	session.RLock()
	defer session.RUnlock()
	var windowSize uint32
	for _, connection := range session.connections {
		windowSize += uint32(connection.windowSize)
	}
	return windowSize
}

func (session *Session) getResendSeq() (uint32, error) {
	var seq uint32
	select {
	case seq = <-session.resendChan:
	case <-session.context.Done():
		return 0, session.context.Err()
	default:
	}
	return seq, nil
}

func (session *Session) getSendSeq() (uint32, error) {
	var seq uint32
	select {
	case seq = <-session.resendChan:
	case seq = <-session.sendChan:
	case <-session.context.Done():
		return 0, session.context.Err()
	}
	return seq, nil
}

func (session *Session) ReceiveWith(localClientID, remoteClientID string, buf []byte) error {
	if session.IsClosed() {
		return ErrSessionClosed
	}

	packet := &pb.Packet{}
	err := proto.Unmarshal(buf, packet)
	if err != nil {
		return err
	}

	if packet.Close {
		return session.handleClosePacket()
	}

	isEstablished := session.IsEstablished()
	if !isEstablished && packet.Handshake {
		return session.handleHandshakePacket(packet)
	}

	session.Lock()
	defer session.Unlock()

	if isEstablished && (len(packet.AckStartSeq) > 0 || len(packet.AckSeqCount) > 0) {
		if len(packet.AckStartSeq) > 0 && len(packet.AckSeqCount) > 0 && len(packet.AckStartSeq) != len(packet.AckSeqCount) {
			return ErrInvalidPacket
		}

		count := 0
		if len(packet.AckStartSeq) > 0 {
			count = len(packet.AckStartSeq)
		} else {
			count = len(packet.AckSeqCount)
		}

		var ackStartSeq, ackEndSeq uint32
		for i := 0; i < count; i++ {
			if len(packet.AckStartSeq) > 0 {
				ackStartSeq = packet.AckStartSeq[i]
			} else {
				ackStartSeq = MinSequenceID
			}

			if len(packet.AckSeqCount) > 0 {
				ackEndSeq = NextSeq(ackStartSeq, int64(packet.AckSeqCount[i]))
			} else {
				ackEndSeq = NextSeq(ackStartSeq, 1)
			}

			if SeqInBetween(session.sendWindowStartSeq, session.sendWindowEndSeq, NextSeq(ackEndSeq, -1)) {
				if !SeqInBetween(session.sendWindowStartSeq, session.sendWindowEndSeq, ackStartSeq) {
					ackStartSeq = session.sendWindowStartSeq
				}
				for seq := ackStartSeq; SeqInBetween(ackStartSeq, ackEndSeq, seq); seq = NextSeq(seq, 1) {
					for key, connection := range session.connections {
						connection.ReceiveAck(seq, key == connKey(localClientID, remoteClientID))
					}
					delete(session.sendWindowData, seq)
				}
				if ackStartSeq == session.sendWindowStartSeq {
					for {
						session.sendWindowStartSeq = NextSeq(session.sendWindowStartSeq, 1)
						if _, ok := session.sendWindowData[session.sendWindowStartSeq]; ok {
							break
						}
						if session.sendWindowStartSeq == session.sendWindowEndSeq {
							break
						}
					}
				}
			}
		}

		session.updateConnWindowSize()
	}

	if isEstablished && packet.BytesRead > session.remoteBytesRead {
		session.remoteBytesRead = packet.BytesRead
		select {
		case session.sendWindowUpdate <- struct{}{}:
		default:
		}
	}

	if isEstablished && packet.SequenceId > 0 {
		if uint32(len(packet.Data)) > session.recvMtu {
			return ErrDataSizeTooLarge
		}

		if CompareSeq(packet.SequenceId, session.recvWindowStartSeq) >= 0 {
			if _, ok := session.recvWindowData[packet.SequenceId]; !ok {
				if session.recvWindowUsed+uint32(len(packet.Data)) > session.recvWindowSize {
					return ErrRecvWindowFull
				}

				session.recvWindowData[packet.SequenceId] = packet.Data
				session.recvWindowUsed += uint32(len(packet.Data))

				if packet.SequenceId == session.recvWindowStartSeq {
					select {
					case session.recvDataUpdate <- struct{}{}:
					default:
					}
				}
			}
		}

		if conn, ok := session.connections[connKey(localClientID, remoteClientID)]; ok {
			conn.SendAck(packet.SequenceId)
		}
	}

	return nil
}

func (session *Session) start() {
	go session.startFlush()
	go session.startCheckBytesRead()

	for _, connection := range session.connections {
		connection.Start()
	}
}

func (session *Session) startFlush() error {
	for {
		select {
		case <-time.After(time.Duration(session.config.FlushInterval) * time.Millisecond):
		case <-session.context.Done():
			return session.context.Err()
		}

		session.RLock()
		shouldFlush := len(session.sendBuffer) > 0
		session.RUnlock()

		if !shouldFlush {
			continue
		}

		err := session.flushSendBuffer()
		if err != nil {
			if session.context.Err() != nil {
				return session.context.Err()
			}
			log.Println(err)
			continue
		}
	}
}

func (session *Session) startCheckBytesRead() error {
	for {
		select {
		case <-time.After(time.Duration(session.config.CheckBytesReadInterval) * time.Millisecond):
		case <-session.context.Done():
			return session.context.Err()
		}

		session.RLock()
		sentTime := session.bytesReadSentTime
		updateTime := session.bytesReadUpdateTime
		bytesRead := session.bytesRead
		session.RUnlock()

		if bytesRead == 0 || sentTime.After(updateTime) || time.Since(updateTime) < time.Duration(session.config.SendBytesReadThreshold)*time.Millisecond {
			continue
		}

		buf, err := proto.Marshal(&pb.Packet{
			BytesRead: bytesRead,
		})
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}

		success := false
		for _, connection := range session.connections {
			err = session.sendWith(connection.localClientID, connection.remoteClientID, buf, connection.RetransmissionTimeout())
			if err != nil {
				log.Println(err)
				time.Sleep(time.Second)
				continue
			}
			success = true
		}

		if success {
			session.updateBytesReadSentTime()
		}
	}
}

func (session *Session) waitForSendWindow(ctx context.Context, n uint32) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	for session.SendWindowUsed()+n > session.sendWindowSize {
		select {
		case <-session.sendWindowUpdate:
		case <-time.After(maxWait):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return session.sendWindowSize - session.SendWindowUsed(), nil
}

func (session *Session) flushSendBuffer() error {
	session.Lock()

	if len(session.sendBuffer) == 0 {
		session.Unlock()
		return nil
	}

	seq := session.sendWindowEndSeq
	buf, err := proto.Marshal(&pb.Packet{
		SequenceId: seq,
		Data:       session.sendBuffer[:len(session.sendBuffer)],
	})
	if err != nil {
		session.Unlock()
		return err
	}

	session.sendWindowData[seq] = buf
	session.sendWindowEndSeq = NextSeq(seq, 1)
	session.sendBuffer = make([]byte, 0, session.sendMtu)

	session.Unlock()

	select {
	case session.sendChan <- seq:
	case <-session.context.Done():
		return session.context.Err()
	}

	return nil
}

func (session *Session) sendHandshakePacket(writeTimeout time.Duration) error {
	buf, err := proto.Marshal(&pb.Packet{
		Handshake:  true,
		ClientIds:  session.localClientIDs,
		WindowSize: session.recvWindowSize,
		Mtu:        session.recvMtu,
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var lock sync.Mutex
	var errMsg []string
	success := make(chan struct{}, 1)
	fail := make(chan struct{}, 1)
	if len(session.connections) > 0 {
		for _, connection := range session.connections {
			wg.Add(1)
			go func(connection *Connection) {
				defer wg.Done()
				err := session.sendWith(connection.localClientID, connection.remoteClientID, buf, writeTimeout)
				if err == nil {
					select {
					case success <- struct{}{}:
					default:
					}
				} else {
					lock.Lock()
					errMsg = append(errMsg, err.Error())
					lock.Unlock()
				}
			}(connection)
		}
	} else {
		for i, localClientID := range session.localClientIDs {
			wg.Add(1)
			remoteClientID := localClientID
			if len(session.remoteClientIDs) > 0 {
				remoteClientID = session.remoteClientIDs[i%len(session.remoteClientIDs)]
			}
			go func(localClientID, remoteClientID string) {
				defer wg.Done()
				err := session.sendWith(localClientID, remoteClientID, buf, writeTimeout)
				if err == nil {
					select {
					case success <- struct{}{}:
					default:
					}
				} else {
					lock.Lock()
					errMsg = append(errMsg, err.Error())
					lock.Unlock()
				}
			}(localClientID, remoteClientID)
		}
	}
	go func() {
		wg.Wait()
		select {
		case fail <- struct{}{}:
		default:
		}
	}()

	select {
	case <-success:
		return nil
	case <-fail:
		return errors.New(strings.Join(errMsg, "; "))
	}
}

func (session *Session) handleHandshakePacket(packet *pb.Packet) error {
	session.Lock()
	defer session.Unlock()

	if session.isEstablished {
		return nil
	}

	if packet.WindowSize == 0 {
		return ErrInvalidPacket
	}
	if packet.WindowSize < session.sendWindowSize {
		session.sendWindowSize = packet.WindowSize
	}

	if packet.Mtu == 0 {
		return ErrInvalidPacket
	}
	if packet.Mtu < session.sendMtu {
		session.sendMtu = packet.Mtu
	}
	session.sendWindowPacketCount = float64(session.sendWindowSize) / float64(session.sendMtu)

	if len(packet.ClientIds) == 0 {
		return ErrInvalidPacket
	}
	n := len(session.localClientIDs)
	if len(packet.ClientIds) < n {
		n = len(packet.ClientIds)
	}

	initialWindowSize := session.sendWindowPacketCount / float64(n)
	connections := make(map[string]*Connection, n)
	for i := 0; i < n; i++ {
		conn, err := NewConnection(session, session.localClientIDs[i], packet.ClientIds[i], initialWindowSize)
		if err != nil {
			return err
		}
		connections[connKey(conn.localClientID, conn.remoteClientID)] = conn
	}
	session.connections = connections

	session.remoteClientIDs = packet.ClientIds
	session.sendChan = make(chan uint32)
	session.resendChan = make(chan uint32, int(session.sendWindowPacketCount)+n)
	session.sendWindowUpdate = make(chan struct{}, 1)
	session.recvDataUpdate = make(chan struct{}, 1)
	session.sendBuffer = make([]byte, 0, session.sendMtu)
	session.sendWindowData = make(map[uint32][]byte)
	session.recvWindowData = make(map[uint32][]byte)
	session.isEstablished = true

	select {
	case session.onAccept <- struct{}{}:
	default:
	}

	return nil
}

func (session *Session) sendClosePacket() error {
	if !session.IsEstablished() {
		return ErrSessionNotEstablished
	}

	buf, err := proto.Marshal(&pb.Packet{
		Close: true,
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var lock sync.Mutex
	var errMsg []string
	success := make(chan struct{}, 1)
	fail := make(chan struct{}, 1)
	for _, connection := range session.connections {
		wg.Add(1)
		go func(connection *Connection) {
			defer wg.Done()
			err := session.sendWith(connection.localClientID, connection.remoteClientID, buf, connection.RetransmissionTimeout())
			if err == nil {
				select {
				case success <- struct{}{}:
				default:
				}
			} else {
				lock.Lock()
				errMsg = append(errMsg, err.Error())
				lock.Unlock()
			}
		}(connection)
	}
	go func() {
		wg.Wait()
		select {
		case fail <- struct{}{}:
		default:
		}
	}()

	select {
	case <-success:
		return nil
	case <-fail:
		return errors.New(strings.Join(errMsg, "; "))
	}
}

func (session *Session) handleClosePacket() error {
	session.readCancel()
	session.writeCancel()
	session.cancel()

	session.Lock()
	session.isClosed = true
	session.Unlock()

	return nil
}

func (session *Session) Dial(ctx context.Context) error {
	session.acceptLock.Lock()
	defer session.acceptLock.Unlock()

	if session.isAccepted {
		return ErrSessionEstablished
	}

	var writeTimeout time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		writeTimeout = time.Until(deadline)
		if writeTimeout < 0 {
			return ctx.Err()
		}
	}

	err := session.sendHandshakePacket(writeTimeout)
	if err != nil {
		return err
	}

	select {
	case <-session.onAccept:
	case <-ctx.Done():
		return ctx.Err()
	}

	go session.start()

	session.isAccepted = true

	return nil
}

func (session *Session) Accept() error {
	session.acceptLock.Lock()
	defer session.acceptLock.Unlock()

	if session.isAccepted {
		return ErrSessionEstablished
	}

	select {
	case <-session.onAccept:
	default:
		return ErrNotHandshake
	}

	go session.start()

	session.isAccepted = true

	return session.sendHandshakePacket(time.Duration(session.config.MaxRetransmissionTimeout) * time.Millisecond)
}

func (session *Session) Read(b []byte) (_ int, e error) {
	defer func() {
		if e == context.DeadlineExceeded {
			e = ErrReadDeadlineExceeded
		}
		if e == context.Canceled {
			e = ErrSessionClosed
		}
	}()

	if session.IsClosed() {
		return 0, ErrSessionClosed
	}

	if !session.IsEstablished() {
		return 0, ErrSessionNotEstablished
	}

	if len(b) == 0 {
		return 0, nil
	}

	session.readLock.Lock()
	defer session.readLock.Unlock()

	for {
		if err := session.readContext.Err(); err != nil {
			return 0, err
		}

		session.RLock()
		_, ok := session.recvWindowData[session.recvWindowStartSeq]
		session.RUnlock()
		if ok {
			break
		}

		select {
		case <-session.recvDataUpdate:
		case <-time.After(maxWait):
		case <-session.readContext.Done():
			return 0, session.readContext.Err()
		}
	}

	session.Lock()
	defer session.Unlock()

	data := session.recvWindowData[session.recvWindowStartSeq]
	if !session.IsStream() && len(b) < len(data) {
		return 0, ErrBufferSizeTooSmall
	}

	bytesReceived := copy(b, data)
	if bytesReceived == len(data) {
		delete(session.recvWindowData, session.recvWindowStartSeq)
		session.recvWindowStartSeq = NextSeq(session.recvWindowStartSeq, 1)
	} else {
		session.recvWindowData[session.recvWindowStartSeq] = data[bytesReceived:]
	}
	session.recvWindowUsed -= uint32(bytesReceived)
	session.bytesRead += uint64(bytesReceived)
	session.bytesReadUpdateTime = time.Now()

	if session.IsStream() {
		for bytesReceived < len(b) {
			data, ok := session.recvWindowData[session.recvWindowStartSeq]
			if !ok {
				break
			}
			n := copy(b[bytesReceived:], data)
			if n == len(data) {
				delete(session.recvWindowData, session.recvWindowStartSeq)
				session.recvWindowStartSeq = NextSeq(session.recvWindowStartSeq, 1)
			} else {
				session.recvWindowData[session.recvWindowStartSeq] = data[n:]
			}
			session.recvWindowUsed -= uint32(n)
			session.bytesRead += uint64(n)
			session.bytesReadUpdateTime = time.Now()
			bytesReceived += n
		}
	}

	return bytesReceived, nil
}

func (session *Session) Write(b []byte) (_ int, e error) {
	defer func() {
		if e == context.DeadlineExceeded {
			e = ErrWriteDeadlineExceeded
		}
		if e == context.Canceled {
			e = ErrSessionClosed
		}
	}()

	if session.IsClosed() {
		return 0, ErrSessionClosed
	}

	if !session.IsEstablished() {
		return 0, ErrSessionNotEstablished
	}

	if !session.IsStream() && (len(b) > int(session.sendMtu) || len(b) > int(session.sendWindowSize)) {
		return 0, ErrDataSizeTooLarge
	}

	if len(b) == 0 {
		return 0, nil
	}

	session.writeLock.Lock()
	defer session.writeLock.Unlock()

	bytesSent := 0
	if session.IsStream() {
		for len(b) > 0 {
			sendWindowAvailable, err := session.waitForSendWindow(session.writeContext, 1)
			if err != nil {
				return bytesSent, err
			}

			n := len(b)
			if n > int(sendWindowAvailable) {
				n = int(sendWindowAvailable)
			}

			session.Lock()
			shouldFlush := sendWindowAvailable == session.sendWindowSize
			c := int(session.sendMtu)
			l := len(session.sendBuffer)
			if n >= c-l {
				n = c - l
				shouldFlush = true
			}
			session.sendBuffer = session.sendBuffer[:l+n]
			copy(session.sendBuffer[l:], b)
			session.bytesWrite += uint64(n)
			bytesSent += n
			session.Unlock()

			if shouldFlush {
				err = session.flushSendBuffer()
				if err != nil {
					return bytesSent, err
				}
			}
			b = b[n:]
		}
	} else {
		_, err := session.waitForSendWindow(session.writeContext, uint32(len(b)))
		if err != nil {
			return bytesSent, err
		}

		session.Lock()
		session.sendBuffer = session.sendBuffer[:len(b)]
		copy(session.sendBuffer, b)
		session.bytesWrite += uint64(len(b))
		bytesSent += len(b)
		session.Unlock()

		err = session.flushSendBuffer()
		if err != nil {
			return bytesSent, err
		}
	}

	return bytesSent, nil
}

func (session *Session) Close() error {
	session.readCancel()
	session.writeCancel()

	timeout := make(chan struct{})
	if session.config.Linger > 0 {
		go func() {
			select {
			case <-time.After(time.Duration(session.config.Linger) * time.Millisecond):
				close(timeout)
			case <-session.context.Done():
			}
		}()
	}

	go func() {
		if session.config.Linger != 0 {
			err := session.flushSendBuffer()
			if err != nil {
				log.Println(err)
			}

			func() {
				for {
					select {
					case <-time.After(100 * time.Millisecond):
						session.RLock()
						isSendFinished := session.sendWindowStartSeq == session.sendWindowEndSeq
						session.RUnlock()
						if isSendFinished {
							return
						}
					case <-timeout:
						return
					}
				}
			}()
		}

		err := session.sendClosePacket()
		if err != nil {
			log.Println(err)
		}

		session.cancel()

		session.Lock()
		session.isClosed = true
		session.Unlock()
	}()

	return nil
}

func (session *Session) LocalAddr() net.Addr {
	return session.localAddr
}

func (session *Session) RemoteAddr() net.Addr {
	return session.remoteAddr
}

func (session *Session) SetDeadline(t time.Time) error {
	err := session.SetReadDeadline(t)
	if err != nil {
		return err
	}
	err = session.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return nil
}

func (session *Session) SetReadDeadline(t time.Time) error {
	if t == zeroTime {
		session.readContext, session.readCancel = context.WithCancel(session.context)
	} else {
		session.readContext, session.readCancel = context.WithDeadline(session.context, t)
	}
	return nil
}

func (session *Session) SetWriteDeadline(t time.Time) error {
	if t == zeroTime {
		session.writeContext, session.writeCancel = context.WithCancel(session.context)
	} else {
		session.writeContext, session.writeCancel = context.WithDeadline(session.context, t)
	}
	return nil
}

// SetLinger sets session linger in unit of millisecond
func (session *Session) SetLinger(t int32) {
	session.config.Linger = t
}

func (session *Session) updateConnWindowSize() {
	totalSize := 0.0
	for _, conn := range session.connections {
		conn.RLock()
		totalSize += float64(conn.windowSize)
		conn.RUnlock()
	}
	if totalSize <= 0 {
		return
	}

	for _, conn := range session.connections {
		conn.Lock()
		n := session.sendWindowPacketCount * (conn.windowSize / totalSize)
		conn.setWindowSize(n)
		conn.Unlock()
	}
}
