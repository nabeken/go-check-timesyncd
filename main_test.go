package main

import (
	"os"
	"testing"
	"time"

	"github.com/nabeken/nagiosplugin/v2"
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
			OutputFn: "normal_output.txt",
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

func TestRunCheck(t *testing.T) {
	type testCase struct {
		OutputFn string
		Expected string
	}

	for _, tc := range []testCase{
		{
			OutputFn: "normal_output.txt",
			Expected: "TIMESYNCD OK: NTP Server: 185.125.190.58 (ntp.ubuntu.com), Offset: 2.3ms, Packet Count: 6, Stratum: 2",
		},
		{
			OutputFn: "warning_offset_output.txt",
			Expected: "TIMESYNCD WARNING: Offset: 60.065ms > 50ms",
		},
		{
			OutputFn: "large_offset_output.txt",
			Expected: "TIMESYNCD CRITICAL: Offset: 41h45m31.391196s > 100ms",
		},
		{
			OutputFn: "large_stratum_output.txt",
			Expected: "TIMESYNCD CRITICAL: Stratum is out of order: 16",
		},
		{
			OutputFn: "not_ok_output.txt",
			Expected: "TIMESYNCD CRITICAL: Packet Count is out of order: 0, Stratum is out of order: 0",
		},
	} {
		t.Run(tc.OutputFn, func(t *testing.T) {
			check := nagiosplugin.NewCheck("TIMESYNCD")

			opts := &_opts{
				Warning:  50 * time.Millisecond,
				Critical: 100 * time.Millisecond,
			}

			runCheck(opts, check, mustParseTimesyncStatus(mustReadTestData(tc.OutputFn)))

			assert.Equal(t, tc.Expected, check.String())
			t.Log(check.String())
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

func mustParseTimesyncStatus(result string) timesyncStatus {
	status, err := parseTimesyncStatus(result)
	if err != nil {
		panic(err)
	}

	return status
}

func mustParseDuration(dur string) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		panic(err)
	}

	return d
}
