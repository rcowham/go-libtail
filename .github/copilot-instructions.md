# go-libtail Copilot Instructions

## Project Overview
Cross-platform Go library for tailing log files with filesystem watching. Originally derived from [fstab/grok_exporter](https://github.com/fstab/grok_exporter) to avoid Oniguruma dependencies and Windows file locking issues.

**Key design goal**: Files are NOT locked on Windows during tailing, allowing log rotation without issues.

## Architecture

### Core Components
- **`tailer/fswatcher`**: Platform-specific filesystem watchers using native APIs
  - Darwin: kqueue (`fswatcher_darwin.go`, separate files for 386/amd64/arm64)
  - Linux: inotify (`fswatcher_linux.go`)
  - Windows: `ReadDirectoryChangesW` (`fswatcher_windows.go`)
  - Fallback: Polling-based watcher (`pollingwatcher.go`)
  
- **`tailer/bufferedTailer.go`**: Wrapper that prevents blocking when lines are processed slowly. Critical for preventing duplicate line reads during log rotation (see detailed comments in file).

- **`tailer/glob`**: Pattern matching for log files. **Constraint**: wildcards only allowed in filenames, not directory paths.

- **`tailer/lineBuffer.go`**: Thread-safe queue using `container/list` with condition variables for blocking/non-blocking operations.

### Key Interfaces
```go
type FileTailer interface {
    Lines() chan *Line
    Errors() chan Error
    Close()
}
```

## Platform-Specific Conventions

### File Organization
- OS-specific files: `fswatcher_{darwin,linux,windows}.go`
- Architecture-specific: `fswatcher_darwin_{386,amd64,arm64}.go`
- Event loops: `fseventProducerLoop_{darwin,linux,windows}.go`
- **No build tags used** - Go automatically selects based on filename patterns

### Implementation Pattern
Each platform implements the `fswatcher` interface:
```go
type fswatcher interface {
    runFseventProducerLoop() fseventProducerLoop
    processEvent(t *fileTailer, event fsevent, log logrus.FieldLogger) Error
    watchDir(path string) (*Dir, Error)
    unwatchDir(dir *Dir) error
    watchFile(file fileMeta) Error
    io.Closer
}
```

## Testing

### Test Execution
```bash
go test ./...                    # Run all tests
go test ./tailer -v              # Verbose output for tailer package
go test ./tailer/fswatcher -v    # Platform-specific tests
```

### Test Structure
- **YAML-based scenario tests**: `tailer/fswatcher_test.go` defines test scenarios in embedded YAML (see `const tests` at top of file)
- Commands: `[mkdir, ...]`, `[log, text, path]`, `[expect, text, path]`, `[logrotate, old, new]`
- Tests cover: single files, globs, multiple directories, rotation, missing files
- **Key test**: `TestShutdownDuringSyscall` - validates graceful shutdown
- Platform-specific behavior tested via conditional logic in `TestAll`

### Critical Edge Cases Tested
1. Log rotation while tailing (move old file, create new with same name)
2. File truncation detection
3. Concurrent writes to multiple log files
4. Missing files at startup (`fail_on_missing_logfile` flag)
5. OSX Finder visibility issues (invisible file detection)

## Dependencies
- **logrus**: Logging (field-based structured logging throughout)
- **prometheus/common**: Metrics (bufferedTailer exposes buffer depth)
- **gopkg.in/yaml.v2**: Test configuration parsing
- **golang.org/x/exp**: Extended Go libraries
- **rcowham/kingpin**: CLI flag parsing (fork with custom modifications)

## Common Workflows

### Adding New Platform Support
1. Create `fswatcher_{os}.go` implementing `fswatcher` interface
2. Create `fseventProducerLoop_{os}.go` for event loop
3. Add platform-specific file metadata in `file_{os}.go`
4. Test with existing YAML scenarios in `fswatcher_test.go`

### Using the Library
```go
import (
    "github.com/rcowham/go-libtail/tailer/fswatcher"
    "github.com/rcowham/go-libtail/tailer/glob"
)

// Create glob pattern
g, _ := glob.FromPath("/var/log/app.log")

// Start tailer
tail, _ := fswatcher.RunFileTailer(
    []glob.Glob{g},
    readall,                  // read existing content
    failOnMissingLogfile,     // error if file doesn't exist
    logger,
)

// Consume lines
for line := range tail.Lines() {
    fmt.Println(line.Line)  // line.File contains source path
}
```

### Debugging Tips
- Use `logger.Level = logrus.DebugLevel` to see filesystem events
- Check `tail.Errors()` channel - errors are non-blocking
- For Windows: verify files aren't locked with `handle.exe` or similar tools
- Polling mode available when native watchers have issues: `RunPollingFileTailer()`

## Anti-Patterns to Avoid
- Don't use globs with wildcards in directory paths - will fail validation
- Don't block on `Lines()` channel without checking `Errors()` - can deadlock
- Don't call `Close()` multiple times - uses channel closure for signaling
- Don't use `filepath.Glob()` directly - use `tailer/glob` package which validates patterns

## Code Style
- Error handling: Custom `Error` type with `Cause()` method for wrapping
- Channels: Unbuffered channels with select statements using `done` channel pattern
- Interfaces: Small, focused interfaces (e.g., `FileTailer`, `lineBuffer`)
- Line readers: Strip Windows `\r` endings automatically in `lineReader.ReadLine()`
