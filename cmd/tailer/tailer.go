// Copyright 2019 Robert Cowham, Perforce Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rcowham/go-libtail/tailer"
	"github.com/rcowham/go-libtail/tailer/fswatcher"
	"github.com/rcowham/go-libtail/tailer/glob"

	"github.com/sirupsen/logrus"
)

var (
	logpath                = flag.String("logpath", "", "Path to the log file to tail.")
	maxLineBytes           = flag.Int("max-line-bytes", 0, "Maximum allowed bytes per line before truncation/error. Values <= 0 use default (1 MiB).")
	maxLines               = flag.Int("max-lines", 0, "Exit after processing this many output lines. 0 means unlimited.")
	noOutput               = flag.Bool("no-output", false, "Do not print lines; useful for throughput testing.")
	decodeEscapedSequences = flag.Bool("decode-escaped-sequences", true, "Decode escaped \\n and \\t sequences in output lines.")
)

type myConfig struct {
	Type                 string
	Path                 string
	PollInterval         time.Duration
	Readall              bool
	FailOnMissingLogfile bool
	MaxLineBytes         int
	MaxLines             int
	NoOutput             bool
	DecodeEscapedSeqs    bool
}

func main() {
	flag.Parse()

	cfg := &myConfig{
		Type:                 "file",
		Path:                 *logpath,
		PollInterval:         0,
		Readall:              true,
		FailOnMissingLogfile: true,
		MaxLineBytes:         *maxLineBytes,
		MaxLines:             *maxLines,
		NoOutput:             *noOutput,
		DecodeEscapedSeqs:    *decodeEscapedSequences,
	}

	tail, err := startTailer(cfg)
	exitOnError(err)

	processed := 0
	startTime := time.Now()

	for {
		select {
		case err, ok := <-tail.Errors():
			if !ok {
				return
			}
			if os.IsNotExist(err.Cause()) {
				exitOnError(fmt.Errorf("error reading log lines: %v: use 'fail_on_missing_logfile: false' in the input configuration if you want grok_exporter to start even though the logfile is missing", err))
			} else {
				exitOnError(fmt.Errorf("error reading log lines: %v", err.Error()))
			}
		case line, ok := <-tail.Lines():
			if !ok {
				return
			}
			processed++
			if !cfg.NoOutput {
				fmt.Fprintf(os.Stdout, "%v\n", formatOutputLine(line.Line, cfg.DecodeEscapedSeqs))
			}
			if cfg.MaxLines > 0 && processed >= cfg.MaxLines {
				tail.Close()
				elapsed := time.Since(startTime)
				if elapsed > 0 {
					fmt.Fprintf(os.Stderr, "processed %d lines in %v (%.2f lines/s)\n", processed, elapsed, float64(processed)/elapsed.Seconds())
				} else {
					fmt.Fprintf(os.Stderr, "processed %d lines\n", processed)
				}
				return
			}
		}
	}
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(-1)
	}
}

func startTailer(cfgInput *myConfig) (fswatcher.FileTailer, error) {
	logger := logrus.New()
	logger.Level = logrus.WarnLevel
	var tail fswatcher.FileTailer
	g, err := glob.FromPath(cfgInput.Path)
	if err != nil {
		return nil, err
	}
	switch {
	case cfgInput.Type == "file":
		options := fswatcher.TailerOptions{
			MaxLineBytes: cfgInput.MaxLineBytes,
		}
		if cfgInput.PollInterval == 0 {
			tail, err = fswatcher.RunFileTailerWithOptions([]glob.Glob{g}, cfgInput.Readall, cfgInput.FailOnMissingLogfile, options, logger)
		} else {
			tail, err = fswatcher.RunPollingFileTailerWithOptions([]glob.Glob{g}, cfgInput.Readall, cfgInput.FailOnMissingLogfile, cfgInput.PollInterval, options, logger)
		}
	case cfgInput.Type == "stdin":
		tail = tailer.RunStdinTailer()
	default:
		return nil, fmt.Errorf("Config error: Input type '%v' unknown.", cfgInput.Type)
	}
	return tail, nil
}

func formatOutputLine(line string, decodeEscapedSeqs bool) string {
	if !decodeEscapedSeqs {
		return line
	}
	line = strings.ReplaceAll(line, `\n`, "\n")
	line = strings.ReplaceAll(line, `\t`, "\t")
	return line
}
