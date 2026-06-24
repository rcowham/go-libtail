# go-libtail

Tailing library based on fstab/grok_exporter tail package - cross compiles easily and doesn't lock Windows files!

Will evolve for use in other libraries such as p4prometheus and go-libp4dlog

**See cmd/tailer for a very simple command line interface showing very basic usage.**

**Consumers of this don't exercise the Windows event version heavily - they use polling tailer for safety!**

## Usage

```go
package main

import (
	"time"

	"github.com/rcowham/go-libtail/tailer/fswatcher"
	"github.com/rcowham/go-libtail/tailer/glob"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	g, _ := glob.FromPath("/var/log/app.log")

	options := fswatcher.TailerOptions{
		MaxLineBytes:       2 * 1024 * 1024, // 2 MiB
		ContinueOnLongLine: true,
	}

	t, err := fswatcher.RunFileTailerWithOptions(
		[]glob.Glob{g},
		true,  // read existing lines on startup
		false, // do not fail if file is missing at startup
		options,
		logger,
	)
	if err != nil {
		panic(err)
	}
	defer t.Close()

	for {
		select {
		case line, ok := <-t.Lines():
			if !ok {
				return
			}
			_ = line
		case terr, ok := <-t.Errors():
			if !ok {
				return
			}
			if terr.Type() == fswatcher.LineTooLong {
				logger.Warnf("skipped oversized line: %v", terr)
				continue
			}
			logger.Errorf("tailer error: %v", terr)
		}
	}
}
```

Use `RunPollingFileTailerWithOptions` if you prefer polling mode:

```go
t, err := fswatcher.RunPollingFileTailerWithOptions(
	[]glob.Glob{g},
	true,
	false,
	10*time.Millisecond,
	options,
	logger,
)
```

## Long Line Handling

- `TailerOptions.MaxLineBytes` sets the maximum allowed bytes per line.
- If `MaxLineBytes <= 0`, the default is 1 MiB.
- When a line exceeds the limit, the emitted error type is `fswatcher.LineTooLong`.
- If `TailerOptions.ContinueOnLongLine` is `false` (default), tailing stops on a long line error.
- If `TailerOptions.ContinueOnLongLine` is `true`, the oversized line is skipped up to the next newline, an error is sent on `Errors()`, and tailing continues.

## Alternatives

Tried [papertrail/go-tail](https://github.com/papertrail/go-tail) but had two problems:

* reopen wasn't working properly
* log files were locked on Windows if being tailed

When I tried to import the files from [fstab/grok_exporter](https://github.com/fstab/grok_exporter) I found the Oniguruma 
dependency made life rather trickier - especially when cross compiling for Windows (even though the instructions all worked - just
took a long time!). In addition that version still relies on the deprecated winfsnotify module.

## ToDo

* Proper config
