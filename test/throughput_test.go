package test

import (
	"fmt"
	"testing"

	mockconn "github.com/nknorg/mockconn-go"
)

// go test -v -run=TestBaseClient
func TestBaseClient(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	tc := &TestCase{id: id, name: fmt.Sprintf("Base case, one client throughput %v packets/s, latency %v, loss %v",
		baseConf.Throughput, baseConf.Latency, baseConf.Loss), bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "base")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestBaseAndLowTpHighLat
func TestBaseAndLowTpHighLat(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	lowTpHighLatConf := baseConf
	lowTpHighLatConf.Throughput = lowThroughput
	lowTpHighLatConf.Latency = highLatency
	tc := &TestCase{id: id, name: fmt.Sprintf("Append 1 client throughput %v packets/s, latency %v, loss %v to base case",
		lowTpHighLatConf.Throughput, lowTpHighLatConf.Latency, lowTpHighLatConf.Loss), bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = lowTpHighLatConf
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "base_and_1_low_tp")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestBaseAnd2LowTpHighLat
func TestBaseAnd2LowTpHighLat(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	lowTpHighLatConf := baseConf
	lowTpHighLatConf.Throughput = lowThroughput
	lowTpHighLatConf.Latency = highLatency
	tc := &TestCase{id: id, name: fmt.Sprintf("Append 2 clients throughput %v packets/s, latency %v, loss %v to base case",
		lowTpHighLatConf.Throughput, lowTpHighLatConf.Latency, lowTpHighLatConf.Loss), bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = lowTpHighLatConf
	tc.mockConfigs["2"] = lowTpHighLatConf
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "base_and_2_low_tp")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestBaseAnd3LowTpHighLat
func TestBaseAnd3LowTpHighLat(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	lowTpHighLatConf := baseConf
	lowTpHighLatConf.Throughput = lowThroughput
	lowTpHighLatConf.Latency = highLatency
	tc := &TestCase{id: id, name: fmt.Sprintf("Append 3 clients throughput %v packets/s, latency %v, loss %v to base case",
		lowTpHighLatConf.Throughput, lowTpHighLatConf.Latency, lowTpHighLatConf.Loss), bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = lowTpHighLatConf
	tc.mockConfigs["2"] = lowTpHighLatConf
	tc.mockConfigs["3"] = lowTpHighLatConf
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "base_and_3_low_tp")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestBaseAnd3LowTpHighLat
func TestBaseAnd4LowTpHighLat(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	lowTpHighLatConf := baseConf
	lowTpHighLatConf.Throughput = lowThroughput
	lowTpHighLatConf.Latency = highLatency
	tc := &TestCase{id: id, name: fmt.Sprintf("Append 4 clients throughput %v packets/s, latency %v, loss %v to base case",
		lowTpHighLatConf.Throughput, lowTpHighLatConf.Latency, lowTpHighLatConf.Loss), bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = lowTpHighLatConf
	tc.mockConfigs["2"] = lowTpHighLatConf
	tc.mockConfigs["3"] = lowTpHighLatConf
	tc.mockConfigs["4"] = lowTpHighLatConf
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "base_and_4_low_tp")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestBaseAnd3DifferTp
func TestBaseAnd3DifferTp(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	low1, low2, low3 := baseConf, baseConf, baseConf
	low1.Throughput = baseConf.Throughput / 2
	low2.Throughput = baseConf.Throughput / 4
	low3.Throughput = baseConf.Throughput / 8

	tc := &TestCase{id: id, name: fmt.Sprintf("Append 3 different throughput %v,%v,%v packets/s, latency %v, loss %v to base case",
		low1.Throughput, low2.Throughput, low3.Throughput, baseConf.Latency, baseConf.Loss), bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = low1
	tc.mockConfigs["2"] = low2
	tc.mockConfigs["3"] = low3
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "base_and_3_diff_tp")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestTpTrendFromAvg
func TestTpTrendFromAvg(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	conf1, conf2 := baseConf, baseConf
	conf1.Throughput = 1024
	conf2.Throughput = 512
	toSend := 80 << 20

	tc := &TestCase{id: id, name: fmt.Sprintf("Two connections, throughputs are %v,%v packets/s, latency %v, loss %v to base case",
		conf1.Throughput, conf2.Throughput, baseConf.Latency, baseConf.Loss), bytesToSend: toSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = conf1
	tc.mockConfigs["1"] = conf2
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "tp_trend_from_avg")
	testResult = append(testResult, tc)
	PrintResult(testResult)
}
