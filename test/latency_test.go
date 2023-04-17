package test

import (
	"fmt"
	"testing"

	mockconn "github.com/nknorg/mockconn-go"
)

// go test -v -run=TestLowAndHighLatency
func TestLowAndHighLatency(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	conf1 := baseConf
	conf1.Latency = lowLatency
	conf2 := baseConf
	conf2.Latency = highLatency
	toSend := 80 << 20

	tc := &TestCase{id: id, name: fmt.Sprintf("Two clients same throughput %v packets/s, low latency %v, high latency %v",
		conf1.Throughput, conf1.Latency, conf2.Latency), bytesToSend: toSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = conf1
	tc.mockConfigs["1"] = conf2
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "low_and_high_latency")
	testResult = append(testResult, tc)

	PrintResult(testResult)
}
