package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/nabeken/nagiosplugin"
)

type _opts struct {
	Warning  time.Duration `short:"w" long:"warning" description:"absolute offset time to result in warning" default:"50ms"`
	Critical time.Duration `short:"c" long:"critical" description:"absolute offset time to result in critical" default:"100ms"`
}

var opts _opts

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			panic(err)
		}
	}

	check := nagiosplugin.NewCheck("TIMESYNCD")
	defer check.Finish()

	// get the timesyncd status
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	output, err := execTimedatectl(ctx, []string{"timesync-status"})
	if err != nil {
		check.Unknownf("failed to execute 'timedatectl timesync-status' command: %s", err)
		return
	}

	status, err := parseTimesyncStatus(output)
	if err != nil {
		check.Unknownf("failed to parse the output from the command: %s", err)
		return
	}

	runCheck(&opts, check, status)
}

func runCheck(opts *_opts, check *nagiosplugin.Check, status timesyncStatus) {
	check.AddResultf(nagiosplugin.OK, "NTP Server: %s (%s)", status.ServerAddress, status.ServerHost)

	absOffset := status.Offset.Abs()

	if absOffset > opts.Critical.Abs() {
		check.AddResultf(nagiosplugin.CRITICAL, "Offset: %s > %s", absOffset, opts.Critical.Abs())
	} else if absOffset > opts.Warning.Abs() {
		check.AddResultf(nagiosplugin.WARNING, "Offset: %s > %s", absOffset, opts.Warning.Abs())
	} else {
		check.AddResultf(nagiosplugin.OK, "Offset: %s", absOffset)
	}

	if status.PacketCount == 0 {
		check.AddResultf(nagiosplugin.CRITICAL, "Packet Count is out of order: %d", status.PacketCount)
	} else {
		check.AddResultf(nagiosplugin.OK, "Packet Count: %d", status.PacketCount)
	}

	if status.Stratum == 0 || status.Stratum == 16 {
		check.AddResultf(nagiosplugin.CRITICAL, "Stratum is out of order: %d", status.Stratum)
	} else {
		check.AddResultf(nagiosplugin.OK, "Stratum: %d", status.Stratum)
	}
}

type timesyncStatus struct {
	ServerHost    string
	ServerAddress string
	Stratum       int
	Offset        time.Duration
	PacketCount   int64
}

func parseTimesyncStatus(result string) (timesyncStatus, error) {
	var status timesyncStatus

	scanner := bufio.NewScanner(strings.NewReader(result))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		values := strings.SplitN(line, ":", 2)
		key := strings.ToLower(values[0])
		value := strings.TrimSpace(values[1])

		switch key {
		case "server":
			status.ServerAddress, status.ServerHost = parseServer(value)

		case "stratum":
			stratum, err := strconv.Atoi(value)
			if err != nil {
				return status, fmt.Errorf("parsing the stratum: %w", err)
			}

			status.Stratum = stratum

		case "packet count":
			pc, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return status, fmt.Errorf("parsing the packet count: %w", err)
			}

			status.PacketCount = pc

		case "offset":
			var offset time.Duration

			// extract the sign
			sign := value[0]
			value = value[1:]

			// split by space
			offsets := strings.SplitN(value, " ", -1)
			for _, v := range offsets {
				// if it has d
				if idx := strings.Index(v, "d"); idx > 0 {
					days, err := strconv.Atoi(v[:idx])
					if err != nil {
						return status, fmt.Errorf("parsing the day in the offset: %w", err)
					}

					offset += time.Duration(days) * 24 * time.Hour
				} else if idx := strings.Index(v, "min"); idx > 0 {
					// replace min with m
					v = v[:idx] + "m"
					dur, err := time.ParseDuration(v)
					if err != nil {
						return status, fmt.Errorf("parsing the minute in the offset: %w", err)
					}
					offset += dur
				} else {
					dur, err := time.ParseDuration(v)
					if err != nil {
						return status, fmt.Errorf("parsing the offset: %w", err)
					}
					offset += dur
				}
			}

			if sign == '-' {
				offset *= -1
			}

			status.Offset = offset
		}

	}

	if err := scanner.Err(); err != nil {
		return status, err
	}

	return status, nil
}

var parseServerRegxp = regexp.MustCompile(`([\d.]+) \(([^\)]+)\)`)

func parseServer(str string) (string, string) {
	matched := parseServerRegxp.FindStringSubmatch(str)
	if len(matched) != 3 {
		return "", ""
	}

	return matched[1], matched[2]
}

// execute the timedatectl command
func execTimedatectl(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, "timedatectl", args...)
	ret, err := cmd.CombinedOutput()
	return string(ret), err
}
