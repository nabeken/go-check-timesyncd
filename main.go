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

var opts struct {
	Warning  time.Duration `short:"w" long:"warning" description:"offset time to result in warning"`
	Critical time.Duration `short:"c" long:"critical" description:"offset time to result in critical"`
}

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
