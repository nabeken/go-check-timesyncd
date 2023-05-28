package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseServer(t *testing.T) {
	gotAddress, gotHost := parseServer("127.0.0.1 (ntp.example.com)")

	if gotAddress != "127.0.0.1" {
		t.Errorf("got '%v', wants 127.0.0.1", gotAddress)
	}

	if gotHost != "ntp.example.com" {
		t.Errorf("got '%v', wants ntp.example.com", gotHost)
	}
}

func TestParseTimesyncStatus(t *testing.T) {
	type testCase struct {
		OutputFn string
		Expected timesyncStatus
	}

	for _, tc := range []testCase{
		{
			OutputFn: "large_offset_output.txt",
			Expected: timesyncStatus{
				ServerHost:    "169.254.169.123",
				ServerAddress: "169.254.169.123",
				Stratum:       3,
				Offset:        mustParseDuration("-41h45m31.391196s"),
				PacketCount:   1,
			},
		},
		{
			OutputFn: "ok_output.txt",
			Expected: timesyncStatus{
				ServerHost:    "ntp.ubuntu.com",
				ServerAddress: "185.125.190.58",
				Stratum:       2,
				Offset:        mustParseDuration("+2.300ms"),
				PacketCount:   6,
			},
		},
		{
			OutputFn: "not_ok_output.txt",
			Expected: timesyncStatus{
				ServerHost:    "169.254.169.124",
				ServerAddress: "169.254.169.124",
				PacketCount:   0,
			},
		},
	} {
		t.Run(tc.OutputFn, func(t *testing.T) {
			actual, err := parseTimesyncStatus(mustReadTestData(tc.OutputFn))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func mustReadTestData(fn string) string {
	b, err := os.ReadFile("_testdata/" + fn)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func mustParseDuration(dur string) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		panic(err)
	}

	return d
}
