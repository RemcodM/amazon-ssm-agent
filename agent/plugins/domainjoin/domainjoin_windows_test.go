// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// +build windows
//
// Package domainjoin implements the domain join plugin.
package domainjoin

import (
	"errors"
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/contracts"
	"github.com/aws/amazon-ssm-agent/agent/jsonutil"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestCase struct {
	Input          DomainJoinPluginInput
	Output         contracts.PluginOutput
	ExecuterErrors []error
	mark           bool
}

const (
	orchestrationDirectory = "OrchesDir"
	s3BucketName           = "bucket"
	s3KeyPrefix            = "key"
	testInstanceID         = "i-12345678"
	bucketRegionErrorMsg   = "AuthorizationHeaderMalformed: The authorization header is malformed; the region 'us-east-1' is wrong; expecting 'us-west-2' status code: 400, request id: []"
	testDirectoryName      = "corp.test.com"
	testDirectoryId        = "d-0123456789"
)

var TestCases = []TestCase{
	generateTestCaseOk(testDirectoryId, testDirectoryName, []string{"10.0.0.0", "10.0.1.0"}),
	generateTestCaseFail(testDirectoryId, testDirectoryName, []string{"10.0.0.2", "10.0.1.2"}),
}

var logger = log.NewMockLog()

func generateTestCaseOk(id string, name string, ipAddress []string) TestCase {

	var out = contracts.PluginOutput{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
		Status:   "Success",
	}

	return TestCase{
		Input:  generateDomainJoinPluginInput(id, name, ipAddress),
		Output: contracts.PluginOutput{out},
		mark:   true,
	}
}

func generateTestCaseFail(id string, name string, ipAddress []string) TestCase {
	var out = contracts.PluginOutput{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 1,
		Status:   "Failed",
	}

	return TestCase{
		Input:  generateDomainJoinPluginInput(id, name, ipAddress),
		Output: contracts.PluginOutput{out},
		mark:   false,
	}
}

func generateDomainJoinPluginInput(id string, name string, ipAddress []string) DomainJoinPluginInput {
	return DomainJoinPluginInput{
		DirectoryId:    id,
		DirectoryName:  name,
		DnsIpAddresses: ipAddress,
	}
}

// TestRunCommands tests the runCommands and runCommandsRawInput methods, which run one set of commands.
func TestRunCommands(t *testing.T) {
	for _, testCase := range TestCases {
		testRunCommands(t, testCase, true)
		testRunCommands(t, testCase, false)
	}
}

// testRunCommands tests the runCommands or the runCommandsRawInput method for one testcase.
func testRunCommands(t *testing.T, testCase TestCase, rawInput bool) {
	logger.On("Error", mock.Anything).Return(nil)
	logger.Infof("test run commands %v", testCase)

	if testCase.mark {
		utilExe = func(log log.T, cmd string, workingDir string, outputRoot string, stdOut string, stdErr string, isAsync bool) (err error) {
			return nil
		}
	} else {
		errCase := errors.New("err here")
		utilExe = func(log log.T, cmd string, workingDir string, outputRoot string, stdOut string, stdErr string, isAsync bool) (err error) {
			return errCase
		}
	}

	makeDir = func(destinationDir string) (err error) {
		return nil
	}
	makeArgs = func(log log.T, pluginInput DomainJoinPluginInput) (commandArguments string) {
		return "cmd"
	}

	var res contracts.PluginOutput
	mockCancelFlag := new(task.MockCancelFlag)
	p := new(Plugin)
	p.StdoutFileName = "stdout"
	p.StderrFileName = "stderr"
	p.MaxStdoutLength = 1000
	p.MaxStderrLength = 1000
	p.OutputTruncatedSuffix = "-more-"
	p.UploadToS3Sync = true
	p.ExecuteUploadOutputToS3Bucket = func(log log.T, pluginID string, orchestrationDir string, outputS3BucketName string, outputS3KeyPrefix string, useTempDirectory bool, tempDir string, Stdout string, Stderr string) []string {
		return []string{}
	}

	if rawInput {
		// prepare plugin input
		var rawPluginInput map[string]interface{}
		err := jsonutil.Remarshal(testCase.Input, &rawPluginInput)
		assert.Nil(t, err)

		res = p.runCommandsRawInput(logger, rawPluginInput, orchestrationDirectory, mockCancelFlag, s3BucketName, s3KeyPrefix, utilExe)
	} else {
		res = p.runCommands(logger, testCase.Input, orchestrationDirectory, mockCancelFlag, s3BucketName, s3KeyPrefix, utilExe)
	}

	assert.Equal(t, testCase.Output, res)
}

// TestMakeArguments tests the makeArguments methods, which build up the command for domainJoin.exe
func TestMakeArguments(t *testing.T) {
	logger.On("Error", mock.Anything).Return(nil)
	getRegion = func() (string, error) {
		return "us-east-1", nil
	}

	domainJoinInput := generateDomainJoinPluginInput(testDirectoryId, testDirectoryName, []string{"172.31.4.141", "172.31.21.240"})
	commandRes, _ := makeArguments(logger, domainJoinInput)
	expected := "./Ec2Config.DomainJoin.exe --directory-id d-0123456789 --directory-name corp.test.com --instance-region us-east-1 --dns-addresses 172.31.4.141 172.31.21.240"

	assert.Equal(t, expected, commandRes)
}
