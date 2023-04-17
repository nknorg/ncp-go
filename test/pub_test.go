package test

import (
	"fmt"
	"time"

	mockconn "github.com/nknorg/mockconn-go"
)

type TestCase struct {
	id          int
	name        string
	numClients  int
	bytesToSend int
	mockConfigs map[string]mockconn.ConnConfig // map localClientID to mock config
	duration    int64
	speed       float64
}

const (
	lowThroughput  = 128  // packets / second
	midThroughput  = 1024 // packets / second
	highThroughput = 2048 // packets / second

	lowLatency  = 50 * time.Millisecond  // ms
	highLatency = 500 * time.Millisecond //ms
	lowLoss     = 0.01                   // 1% loss
	highLoss    = 0.1                    // 10% loss

	bytesToSend = 8 << 20
)

var baseConf = mockconn.ConnConfig{Addr1: "Alice", Addr2: "Bob", Throughput: midThroughput, Latency: lowLatency,
	WriteTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second}

var testResult []*TestCase
var id = 0

func run(tc *TestCase, imgName string) *TestCase {
	fmt.Printf("\n>>>>>> Case %v, %v\n", id, tc.name)

	var ts TestSession
	ts.Create(tc.mockConfigs, tc.numClients)
	ts.DialUp()

	writeChan := make(chan int64, 3)
	readChan := make(chan int64, 3)
	go ts.write(ts.localSess, tc.bytesToSend, writeChan)
	go ts.read(ts.remoteSess, readChan)

	bytesReceived := <-readChan
	count := <-readChan
	ms := <-readChan

	speed := float64(bytesReceived) / (1 << 20) / (float64(ms) / 1000.0)
	throughput := float64(count) / (float64(ms) / 1000.0)

	fmt.Printf("\n%v received %v bytes at %.3f MB/s, throughput:%.1f packets/s, duration: %v ms \n",
		ts.remoteSess.LocalAddr(), bytesReceived, speed, throughput, ms)

	tc.duration = ms
	tc.speed = speed

	return tc
}

// Print test result to compare previous version and new version
func PrintResult(testResult []*TestCase) {
	fmt.Printf("\nid \tconn \tspeed \t\tduration \ttest case description\n")
	for _, tc := range testResult {
		fmt.Printf("%v \t%v \t%.3f MB/s \t%v ms \t%v \n",
			tc.id, tc.numClients, tc.speed, tc.duration, tc.name)
	}
	fmt.Println()
}
