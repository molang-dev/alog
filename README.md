# alog

`alog` is a small Go logging library with console output, daily file output,
package-level helper functions, and independent logger instances.

## Features

- Console logging.
- File logging.
- Daily log files named like `2026-06-27.log`.
- Existing daily log files are appended to, not overwritten.
- The active log file is checked on every print; when the local date changes,
  the logger closes the old file and opens the new date file.
- Console and file output can be enabled at the same time.
- Empty optional fields are omitted and do not produce empty `|` columns.
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
	alog.I("Startup", "hello %s", "world")
}
```

Example output:

```text
2026-06-27 13:55:34.386|I|12345|Startup|myapp|hello world
```

## Logger Instances

Use `alog.New()` when you want a logger with independent configuration.

```go
logger := alog.New()
logger.SetPrefix("Worker")
logger.SetLevel(alog.LevelDebug)
logger.D("Sync", "loaded %d items", 10)
```

With a prefix, output looks like:

```text
2026-06-27 13:55:34.386|D|12345|Worker|Sync|myapp|loaded 10 items
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

## Format

The full format is:

```text
YYYY-MM-DD HH:mm:ss.SSS|Level|PID(TID)|Prefix|Tag|PackageName|Message
```

`TID` and `Prefix` are optional. When an optional field is empty, it is omitted
and does not reserve a `|` column.

The current implementation does not emit `TID`, so the process field is printed
as `PID`.

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
