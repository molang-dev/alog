package alog

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLoggerFormat(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.I("Tag", "hello %s", "world")

	line := strings.TrimSpace(out.String())
	pattern := `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}\|I\|\d+\|Tag\|hello world$`
	if !regexp.MustCompile(pattern).MatchString(line) {
		t.Fatalf("unexpected line: %q", line)
	}
	if strings.Contains(line, "||") {
		t.Fatalf("line contains empty field separator: %q", line)
	}
}

func TestLoggerLevelFiltersLowLevels(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.SetLevel(LevelInfo)
	log.D("Tag", "debug")
	log.I("Tag", "info")

	text := out.String()
	if strings.Contains(text, "debug") {
		t.Fatalf("debug log should be filtered: %q", text)
	}
	if !strings.Contains(text, "info") {
		t.Fatalf("info log should be printed: %q", text)
	}
}

func TestLoggerErrorCannotBeFiltered(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.SetLevel(LevelFatal)
	log.E("Tag", "error")

	if !strings.Contains(out.String(), "error") {
		t.Fatalf("error log should not be filtered: %q", out.String())
	}
}

func TestLoggerWritesDateFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	log := New()
	log.SetFlags(FlagFile)
	log.I("Tag", "file message")

	name := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.log$`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !name.MatchString(entry.Name()) {
			continue
		}
		content, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "file message") {
			return
		}
	}

	t.Fatal("date log file was not written")
}

func TestLoggerAppendsDateFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	log := New()
	log.SetFlags(FlagFile)
	log.I("Tag", "first")
	log.I("Tag", "second")

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one log file, got %d", len(entries))
	}

	content, err := os.ReadFile(entries[0].Name())
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "first") || !strings.Contains(text, "second") {
		t.Fatalf("log file should contain appended messages: %q", text)
	}
}

func TestLoggerWritesDateFileInDirWithPrefix(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "logs")

	log := New()
	log.SetDir(dir)
	log.SetFilePrefix("app-")
	log.SetFlags(FlagFile)
	log.I("Tag", "prefixed file")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one log file, got %d", len(entries))
	}
	if !regexp.MustCompile(`^app-\d{4}-\d{2}-\d{2}\.log$`).MatchString(entries[0].Name()) {
		t.Fatalf("unexpected log file name: %q", entries[0].Name())
	}

	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "prefixed file") {
		t.Fatalf("log file should contain message: %q", string(content))
	}
}

func TestLoggerChangingFilePrefixSwitchesFile(t *testing.T) {
	dir := t.TempDir()

	log := New()
	log.SetDir(dir)
	log.SetFilePrefix("first-")
	log.SetFlags(FlagFile)
	log.I("Tag", "first message")
	log.SetFilePrefix("second-")
	log.I("Tag", "second message")

	firstFiles, err := filepath.Glob(filepath.Join(dir, "first-*.log"))
	if err != nil {
		t.Fatal(err)
	}
	secondFiles, err := filepath.Glob(filepath.Join(dir, "second-*.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(firstFiles) != 1 || len(secondFiles) != 1 {
		t.Fatalf("expected one first and one second file, got %d and %d", len(firstFiles), len(secondFiles))
	}

	firstContent, err := os.ReadFile(firstFiles[0])
	if err != nil {
		t.Fatal(err)
	}
	secondContent, err := os.ReadFile(secondFiles[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(firstContent), "first message") {
		t.Fatalf("first file should contain first message: %q", string(firstContent))
	}
	if !strings.Contains(string(secondContent), "second message") {
		t.Fatalf("second file should contain second message: %q", string(secondContent))
	}
}

func TestLoggerCallerFlagsStartAtConfiguredLevel(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.SetCallerFlags(LevelWarning, FlagShortFile|FlagFunc)

	log.I("Tag", "info")
	infoLine := strings.TrimSpace(out.String())
	if strings.Contains(infoLine, "alog_test.go:") || strings.Contains(infoLine, "TestLoggerCallerFlagsStartAtConfiguredLevel") {
		t.Fatalf("info log should not include caller fields: %q", infoLine)
	}

	out.Reset()
	log.W("Tag", "warn")
	warnLine := strings.TrimSpace(out.String())
	if !strings.Contains(warnLine, "alog_test.go:") {
		t.Fatalf("warning log should include short file: %q", warnLine)
	}
	if !strings.Contains(warnLine, "TestLoggerCallerFlagsStartAtConfiguredLevel") {
		t.Fatalf("warning log should include function name: %q", warnLine)
	}
}

func TestLoggerLongFileWinsOverShortFile(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.SetCallerFlags(LevelVerbose, FlagShortFile|FlagLongFile)
	log.V("Tag", "verbose")

	line := strings.TrimSpace(out.String())
	if !regexp.MustCompile(`/alog_test\.go:\d+`).MatchString(line) {
		t.Fatalf("log should include long file path: %q", line)
	}
}
