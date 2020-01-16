package ncp

import (
	"math"
	"time"
)

var (
	zeroTime time.Time
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

func PrevSeq(seq, step uint32) uint32 {
	return (seq-MinSequenceID-step)%(math.MaxUint32-MinSequenceID+1) + MinSequenceID
}

func NextSeq(seq, step uint32) uint32 {
	return (seq-MinSequenceID+step)%(math.MaxUint32-MinSequenceID+1) + MinSequenceID
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
