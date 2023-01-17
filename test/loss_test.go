package test

import (
	"fmt"
	"testing"

	mockconn "github.com/nknorg/mockconn-go"
)

// go test -v -run=TestLowAndHighLoss
func TestLowAndHighLoss(t *testing.T) {
	if testResult == nil {
		testResult = make([]*TestCase, 0)
	}

	id++
	conf1, conf2 := baseConf, baseConf
	conf1.Loss = 0.01
	conf2.Loss = 0.1
	toSend := 8 << 20

	tc := &TestCase{id: id, name: fmt.Sprintf("Base case with two connections with low loss %v, high loss %v",
		conf1.Loss, conf2.Loss), bytesToSend: toSend}
	tc.mockConfigs = make(map[string]mockconn.ConnConfig)
	tc.mockConfigs["0"] = baseConf
	tc.mockConfigs["1"] = conf1
	tc.mockConfigs["2"] = conf2
	tc.numClients = len(tc.mockConfigs)

	tc = run(tc, "Base_and_low_and_high_loss")
	testResult = append(testResult, tc)

	PrintResult(testResult)
}
