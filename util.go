package ncp

import (
	"errors"
	"math"
	"time"
)

var (
	zeroTime   time.Time
	maxWait    = time.Second
	errMaxWait = errors.New("max wait time reached")
)

type SeqHeap []uint32

func (h SeqHeap) Len() int           { return len(h) }
func (h SeqHeap) Less(i, j int) bool { return CompareSeq(h[i], h[j]) < 0 }
func (h SeqHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *SeqHeap) Push(x interface{}) {
	*h = append(*h, x.(uint32))
}

func (h *SeqHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func NextSeq(seq uint32, step int64) uint32 {
	var max int64 = math.MaxUint32 - MinSequenceID + 1
	res := (int64(seq) - MinSequenceID + step) % max
	if res < 0 {
		res += max
	}
	return uint32(res + MinSequenceID)
}

func SeqInBetween(startSeq, endSeq, targetSeq uint32) bool {
	if startSeq <= endSeq {
		return targetSeq >= startSeq && targetSeq < endSeq
	}
	return targetSeq >= startSeq || targetSeq < endSeq
}

func CompareSeq(seq1, seq2 uint32) int {
	if seq1 == seq2 {
		return 0
	}
	if seq1 < seq2 {
		if seq2-seq1 < math.MaxUint32/2 {
			return -1
		}
		return 1
	}
	if seq1-seq2 < math.MaxUint32/2 {
		return 1
	}
	return -1
}

func connKey(localClientID, remoteClientID string) string {
	return localClientID + " - " + remoteClientID
}
