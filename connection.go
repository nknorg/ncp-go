package ncp

import (
	"container/heap"
	"context"
	"log"
	"math"
	"sync"
	"time"

	"github.com/nknorg/ncp-go/pb"
	"google.golang.org/protobuf/proto"
)

type Connection struct {
	session          *Session
	localClientID    string
	remoteClientID   string
	windowSize       float64
	sendWindowUpdate chan struct{}

	sync.RWMutex
	timeSentSeq           map[uint32]time.Time
	resentSeq             map[uint32]struct{}
	sendAckQueue          SeqHeap
	retransmissionTimeout time.Duration
}

func NewConnection(session *Session, localClientID, remoteClientID string, initialWindowSize float64) (*Connection, error) {
	conn := &Connection{
		session:               session,
		localClientID:         localClientID,
		remoteClientID:        remoteClientID,
		windowSize:            initialWindowSize,
		retransmissionTimeout: time.Duration(session.config.InitialRetransmissionTimeout) * time.Millisecond,
		sendWindowUpdate:      make(chan struct{}, 1),
		timeSentSeq:           make(map[uint32]time.Time),
		resentSeq:             make(map[uint32]struct{}),
		sendAckQueue:          make(SeqHeap, 0),
	}
	heap.Init(&conn.sendAckQueue)
	return conn, nil
}

func (conn *Connection) SendWindowUsed() uint32 {
	conn.RLock()
	defer conn.RUnlock()
	return uint32(len(conn.timeSentSeq))
}

func (conn *Connection) RetransmissionTimeout() time.Duration {
	conn.RLock()
	defer conn.RUnlock()
	return conn.retransmissionTimeout
}

func (conn *Connection) SendAck(sequenceID uint32) {
	conn.Lock()
	heap.Push(&conn.sendAckQueue, sequenceID)
	conn.Unlock()
}

func (conn *Connection) SendAckQueueLen() int {
	conn.RLock()
	defer conn.RUnlock()
	return conn.sendAckQueue.Len()
}

func (conn *Connection) ReceiveAck(sequenceID uint32, isSentByMe bool) {
	conn.Lock()
	defer conn.Unlock()

	t, ok := conn.timeSentSeq[sequenceID]
	if !ok {
		return
	}

	if _, ok := conn.resentSeq[sequenceID]; !ok {
		conn.setWindowSize(conn.windowSize + 1)
	}

	if isSentByMe {
		rtt := time.Since(t)
		conn.retransmissionTimeout += time.Duration(math.Tanh(float64(3*rtt-conn.retransmissionTimeout)/float64(time.Millisecond)/1000) * 100 * float64(time.Millisecond))
		if conn.retransmissionTimeout > time.Duration(conn.session.config.MaxRetransmissionTimeout)*time.Millisecond {
			conn.retransmissionTimeout = time.Duration(conn.session.config.MaxRetransmissionTimeout) * time.Millisecond
		}
	}

	delete(conn.timeSentSeq, sequenceID)
	delete(conn.resentSeq, sequenceID)

	select {
	case conn.sendWindowUpdate <- struct{}{}:
	default:
	}
}

func (conn *Connection) waitForSendWindow(ctx context.Context) error {
	for float64(conn.SendWindowUsed()) >= conn.windowSize {
		select {
		case <-conn.sendWindowUpdate:
		case <-time.After(maxWait):
			return errMaxWait
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (conn *Connection) Start() {
	go conn.tx()
	go conn.sendAck()
	go conn.checkTimeout()
}

func (conn *Connection) tx() error {
	var seq uint32
	var err error
	for {
		if seq == 0 {
			seq, err = conn.session.getResendSeq()
			if err != nil {
				return err
			}
		}

		if seq == 0 {
			err = conn.waitForSendWindow(conn.session.context)
			if err == errMaxWait {
				continue
			}
			if err != nil {
				return err
			}

			seq, err = conn.session.getSendSeq()
			if err != nil {
				return err
			}
		}

		buf := conn.session.GetDataToSend(seq)
		if len(buf) == 0 {
			conn.Lock()
			delete(conn.timeSentSeq, seq)
			delete(conn.resentSeq, seq)
			conn.Unlock()
			seq = 0
			continue
		}

		err = conn.session.sendWith(conn.localClientID, conn.remoteClientID, buf, conn.retransmissionTimeout)
		if err != nil {
			if conn.session.IsClosed() {
				return ErrSessionClosed
			}
			if err == ErrConnClosed {
				return err
			}
			if conn.session.config.Verbose {
				log.Println(err)
			}

			// reduce window size
			conn.Lock()
			conn.setWindowSize(conn.windowSize / 2)
			conn.Unlock()
			conn.session.updateConnWindowSize()

			select {
			case conn.session.resendChan <- seq:
				seq = 0
			case <-conn.session.context.Done():
				return conn.session.context.Err()
			}
			time.Sleep(time.Second)
			continue
		}

		conn.Lock()
		if _, ok := conn.timeSentSeq[seq]; !ok {
			conn.timeSentSeq[seq] = time.Now()
		}
		delete(conn.resentSeq, seq)
		conn.Unlock()

		seq = 0
	}
}

func (conn *Connection) sendAck() error {
	for {
		select {
		case <-time.After(time.Duration(conn.session.config.SendAckInterval) * time.Millisecond):
		case <-conn.session.context.Done():
			return conn.session.context.Err()
		}

		if conn.SendAckQueueLen() == 0 {
			continue
		}

		ackStartSeqList := make([]uint32, 0)
		ackSeqCountList := make([]uint32, 0)

		conn.Lock()
		for conn.sendAckQueue.Len() > 0 && len(ackStartSeqList) < int(conn.session.config.MaxAckSeqListSize) {
			ackStartSeq := heap.Pop(&conn.sendAckQueue).(uint32)
			ackSeqCount := uint32(1)
			for conn.sendAckQueue.Len() > 0 && conn.sendAckQueue[0] == NextSeq(ackStartSeq, int64(ackSeqCount)) {
				heap.Pop(&conn.sendAckQueue)
				ackSeqCount++
			}

			ackStartSeqList = append(ackStartSeqList, ackStartSeq)
			ackSeqCountList = append(ackSeqCountList, ackSeqCount)
		}
		conn.Unlock()

		omitCount := true
		for _, c := range ackSeqCountList {
			if c != 1 {
				omitCount = false
				break
			}
		}
		if omitCount {
			ackSeqCountList = nil
		}

		buf, err := proto.Marshal(&pb.Packet{
			AckStartSeq: ackStartSeqList,
			AckSeqCount: ackSeqCountList,
			BytesRead:   conn.session.GetBytesRead(),
		})
		if err != nil {
			if conn.session.config.Verbose {
				log.Println(err)
			}
			time.Sleep(time.Second)
			continue
		}

		err = conn.session.sendWith(conn.localClientID, conn.remoteClientID, buf, conn.retransmissionTimeout)
		if err != nil {
			if conn.session.IsClosed() {
				return ErrSessionClosed
			}
			if err == ErrConnClosed {
				return err
			}
			if conn.session.config.Verbose {
				log.Println(err)
			}
			time.Sleep(time.Second)
			continue
		}

		conn.session.updateBytesReadSentTime()
	}
}

func (conn *Connection) checkTimeout() error {
	for {
		select {
		case <-time.After(time.Duration(conn.session.config.CheckTimeoutInterval) * time.Millisecond):
		case <-conn.session.context.Done():
			return conn.session.context.Err()
		}

		threshold := time.Now().Add(-conn.retransmissionTimeout)
		conn.Lock()
		newResend := false
		for seq, t := range conn.timeSentSeq {
			if _, ok := conn.resentSeq[seq]; ok {
				continue
			}
			if t.Before(threshold) {
				select {
				case conn.session.resendChan <- seq:
					conn.resentSeq[seq] = struct{}{}
					conn.setWindowSize(conn.windowSize / 2)
					newResend = true
				case <-conn.session.context.Done():
					return conn.session.context.Err()
				}
			}
		}
		conn.Unlock()
		if newResend {
			conn.session.updateConnWindowSize()
		}
	}
}

func (conn *Connection) setWindowSize(n float64) {
	if n < float64(conn.session.config.MinConnectionWindowSize) {
		n = float64(conn.session.config.MinConnectionWindowSize)
	}
	conn.windowSize = n
}
