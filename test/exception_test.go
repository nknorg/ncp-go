package test

import (
	"fmt"
	"testing"
	"time"

	mockconn "github.com/nknorg/mockconn-go"
)

// go test -v -run=TestCloseRemoteRead
func TestCloseRemoteRead(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	tc := &TestCase{id: id, name: "Two connections, the second one will close remote read in 2 seconds", bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	baseConf.WriteTimeout = 3 * time.Second
	baseConf.ReadTimeout = 3 * time.Second
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = baseConf
	tc.numClients = len(tc.mockConfigs)

	tc = closeRemoteRead(tc)
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestCloseRemoteWrite
func TestCloseRemoteWrite(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	tc := &TestCase{id: id, name: "Two connections, the second one will close write in 2 seconds", bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	baseConf.WriteTimeout = 3 * time.Second
	baseConf.ReadTimeout = 3 * time.Second
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = baseConf
	tc.numClients = len(tc.mockConfigs)

	tc = closeRemoteWrite(tc)
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestCloseLocalRead
func TestCloseLocalRead(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	tc := &TestCase{id: id, name: "Two connections, the second one will close remote read in 2 seconds", bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	baseConf.WriteTimeout = 3 * time.Second
	baseConf.ReadTimeout = 3 * time.Second
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = baseConf
	tc.numClients = len(tc.mockConfigs)

	tc = closeLocalRead(tc)
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

// go test -v -run=TestCloseLocalWrite
func TestCloseLocalWrite(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	tc := &TestCase{id: id, name: "Two connections, the second one will close write in 2 seconds", bytesToSend: bytesToSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	baseConf.WriteTimeout = 3 * time.Second
	baseConf.ReadTimeout = 3 * time.Second
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = baseConf
	tc.numClients = len(tc.mockConfigs)

	tc = closeLocalWrite(tc)
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

func closeRemoteRead(tc *TestCase) *TestCase {
	fmt.Printf("\n>>>>>> Case %v, %v\n", id, tc.name)

	var ts TestSession
	ts.Create(tc.mockConfigs, tc.numClients)
	ts.DialUp()

	writeChan := make(chan int64, 3)
	readChan := make(chan int64, 3)
	go ts.write(ts.localSess, tc.bytesToSend, writeChan)
	go ts.read(ts.remoteSess, readChan)

	time.Sleep(2 * time.Second)
	ts.CloseOneConnectionRead(ts.remoteSess, "1")

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

// go test -v -run=TestCloseAndOpenRemoteRead
func TestCloseAndOpenRemoteRead(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	toSend := 80 << 20
	tc := &TestCase{id: id, name: "Two connections, the second one will close remote read in 2 seconds", bytesToSend: toSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	baseConf.WriteTimeout = 3 * time.Second
	baseConf.ReadTimeout = 3 * time.Second
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = baseConf
	tc.numClients = len(tc.mockConfigs)

	tc = pauseAndResumeRemoteRead(tc)
	testResult = append(testResult, tc)
	PrintResult(testResult)
}

func pauseAndResumeRemoteRead(tc *TestCase) *TestCase {
	fmt.Printf("\n>>>>>> Case %v, %v\n", id, tc.name)

	var ts TestSession
	ts.Create(tc.mockConfigs, tc.numClients)
	ts.DialUp()

	writeChan := make(chan int64, 3)
	readChan := make(chan int64, 3)
	go ts.write(ts.localSess, tc.bytesToSend, writeChan)
	go ts.read(ts.remoteSess, readChan)

	time.Sleep(5 * time.Second)
	ts.PauseOneConnectionRead(ts.remoteSess, "1")
	time.Sleep(1 * time.Second)
	ts.ResumeOneConnectionRead(ts.remoteSess, "1")

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

func closeLocalRead(tc *TestCase) *TestCase {
	fmt.Printf("\n>>>>>> Case %v, %v\n", id, tc.name)

	var ts TestSession
	ts.Create(tc.mockConfigs, tc.numClients)
	ts.DialUp()

	writeChan := make(chan int64, 3)
	readChan := make(chan int64, 3)
	go ts.write(ts.localSess, tc.bytesToSend, writeChan)
	go ts.read(ts.remoteSess, readChan)

	time.Sleep(2 * time.Second)
	ts.CloseOneConnectionRead(ts.remoteSess, "1")

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

func closeRemoteWrite(tc *TestCase) *TestCase {
	fmt.Printf("\n>>>>>> Case %v, %v\n", id, tc.name)

	var ts TestSession
	ts.Create(tc.mockConfigs, tc.numClients)
	ts.DialUp()

	writeChan := make(chan int64, 3)
	readChan := make(chan int64, 3)
	go ts.write(ts.localSess, tc.bytesToSend, writeChan)
	go ts.read(ts.remoteSess, readChan)

	time.Sleep(2 * time.Second)
	ts.CloseOneConnectionWrite(ts.remoteSess, "1")

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

func closeLocalWrite(tc *TestCase) *TestCase {
	fmt.Printf("\n>>>>>> Case %v, %v\n", id, tc.name)

	var ts TestSession
	ts.Create(tc.mockConfigs, tc.numClients)
	ts.DialUp()

	writeChan := make(chan int64, 3)
	readChan := make(chan int64, 3)
	go ts.write(ts.localSess, tc.bytesToSend, writeChan)
	go ts.read(ts.remoteSess, readChan)

	time.Sleep(2 * time.Second)
	ts.CloseOneConnectionWrite(ts.localSess, "1")

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
