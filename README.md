# alog

`alog` is a small Go logging library with console output, daily file output,
package-level helper functions, and independent logger instances.

## Features

- Console logging.
- File logging.
- Daily log files named like `2026-06-27.log`.
- Configurable file output directory.
- Configurable daily file prefix, such as `app-2026-06-27.log`.
- Existing daily log files are appended to, not overwritten.
- The active log file is checked on every print; when the local date changes,
  the logger closes the old file and opens the new date file.
- Console and file output can be enabled at the same time.
- Empty optional fields are omitted and do not produce empty `|` columns.
- Optional caller fields can be enabled only at or above a configured level.
- Package-level helpers such as `alog.I(...)`.
- Independent logger instances through `alog.New()`.
- ANSI colors for console output:
  - Verbose: white
  - Debug: blue
  - Info: green
  - Warning: yellow
  - Error and Fatal: red

## Installation

```bash
go get github.com/molang-dev/alog
```

## Quick Start

```go
package main

import "github.com/molang-dev/alog"

func main() {
	alog.V("Startup", "verbose message")
	alog.D("Startup", "debug message")
	alog.I("Startup", "info message")
	alog.W("Startup", "warning message")
	alog.E("Startup", "error message")
	alog.Fatal("Startup", "fatal message and exit")
}
```

Example output:

<span style="color: white">2026-06-27 13:55:34.386|V|12345|Startup|verbose message</span><br>
<span style="color: blue">2026-06-27 13:55:34.386|D|12345|Startup|debug message</span><br>
<span style="color: green">2026-06-27 13:55:34.386|I|12345|Startup|info message</span><br>
<span style="color: goldenrod">2026-06-27 13:55:34.386|W|12345|Startup|warning message</span><br>
<span style="color: red">2026-06-27 13:55:34.386|E|12345|Startup|error message</span><br>
<span style="color: red">2026-06-27 13:55:34.386|F|12345|Startup|fatal message and exit</span>

## Logger Instances

Use `alog.New()` when you want a logger with independent configuration.

```go
logger := alog.New()
logger.SetLevel(alog.LevelDebug)
logger.D("Sync", "loaded %d items", 10)
```

Output looks like:

```text
2026-06-27 13:55:34.386|D|12345|Sync|loaded 10 items
```

## Output Flags

Flags control where logs are written.

```go
logger := alog.New()
logger.SetFlags(alog.FlagScreen | alog.FlagFile | alog.FlagColor)
logger.I("Main", "screen and file")
```

Available flags:

- `alog.FlagScreen`: write to the configured screen writer.
- `alog.FlagFile`: write to the daily date file.
- `alog.FlagColor`: colorize screen output with ANSI colors.

The default flags are:

```go
alog.FlagScreen | alog.FlagColor
```

## File Output

Enable file output with `FlagFile`.

```go
logger := alog.New()
logger.SetFlags(alog.FlagFile)
logger.I("File", "written to today's log file")
```

The file name uses the current local date:

```text
2026-06-27.log
```

If the file already exists, new logs are appended. The date is checked every
time a log is printed, so a long-running process automatically switches to a
new file after midnight.

Set a directory with `SetDir`:

```go
logger := alog.New()
logger.SetDir("./logs")
logger.SetFlags(alog.FlagFile)
logger.I("File", "written to ./logs")
```

This writes to:

```text
./logs/2026-06-27.log
```

The directory is created when the first file log is written.

Set a file prefix with `SetFilePrefix`:

```go
logger := alog.New()
logger.SetFilePrefix("app-")
logger.SetFlags(alog.FlagFile)
logger.I("File", "written to a prefixed file")
```

This writes to:

```text
app-2026-06-27.log
```

Use both together:

```go
logger := alog.New()
logger.SetDir("./logs")
logger.SetFilePrefix("app-")
logger.SetFlags(alog.FlagScreen | alog.FlagFile)
logger.I("File", "screen and ./logs/app-2026-06-27.log")
```

Changing the directory or file prefix closes the old file. The next file log
opens the new target and continues appending if that file already exists.

## Format

The default format is:

```text
YYYY-MM-DD HH:mm:ss.SSS|Level|PID(TID)|Tag|Message
```

`TID` and caller fields are optional. When an optional field is empty, it is
omitted and does not reserve a `|` column.

The current implementation does not emit `TID`, so the process field is printed
as `PID`.

When caller fields are enabled, they are inserted after `Tag` and before
`Message`:

```text
YYYY-MM-DD HH:mm:ss.SSS|Level|PID|Tag|File:Line|Func|Message
```

## Caller Fields

Caller fields are disabled by default because they use `runtime.Caller`.

Use `SetCallerFlags` to enable caller fields at or above a minimum level:

```go
logger := alog.New()
logger.SetCallerFlags(alog.LevelWarning, alog.FlagShortFile|alog.FlagFunc)
```

With this configuration, verbose, debug, and info logs do not include caller
fields. Warning, error, and fatal logs include them:

```text
2026-06-27 13:55:34.386|W|12345|Sync|main.go:23|main.main|slow response
```

Available caller flags:

- `alog.FlagShortFile`: add the caller file base name and line, such as
  `main.go:23`.
- `alog.FlagLongFile`: add the caller full file path and line, such as
  `/app/main.go:23`.
- `alog.FlagFunc`: add the caller function name, such as `main.main`.

If `FlagShortFile` and `FlagLongFile` are both set, `FlagLongFile` is used.

The package-level default logger also supports caller configuration:

```go
alog.SetCallerFlags(alog.LevelError, alog.FlagLongFile|alog.FlagFunc)
```

## Levels

Levels are ordered from low to high:

```text
Verbose < Debug < Info < Warning < Error < Fatal
```

Use `SetLevel` to filter logs below the configured level:

```go
logger := alog.New()
logger.SetLevel(alog.LevelInfo)
logger.D("Debug", "this is filtered")
logger.I("Info", "this is printed")
```

Error and fatal logs are always printed, even when the configured level is
higher than them.

## Timing

Use `Time` and `TimeEnd` to measure elapsed time. `TimeEnd` writes a debug log.

```go
id := alog.Time()
// do work
alog.TimeEnd(id, "Work", "finished")
```

Example message suffix:

```text
elapsed=12.3ms
```

## Fatal Logs

`Fatal` writes the fatal log, writes the current stack trace, and exits the
process with status code `1`.

```go
alog.Fatal("Main", "cannot continue: %v", err)
```
