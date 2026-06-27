package alog

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLoggerFormatSkipsEmptyPrefix(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.I("Tag", "hello %s", "world")

	line := strings.TrimSpace(out.String())
	pattern := `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}\|I\|\d+\|Tag\|` + regexp.QuoteMeta(filepath.Base(os.Args[0]))
	if !regexp.MustCompile(pattern).MatchString(line) {
		t.Fatalf("unexpected line: %q", line)
	}
	if strings.Contains(line, "||") {
		t.Fatalf("line contains empty field separator: %q", line)
	}
}

func TestLoggerFormatIncludesPrefix(t *testing.T) {
	var out bytes.Buffer
	log := New()
	log.SetOutput(&out)
	log.SetFlags(FlagScreen)
	log.SetPrefix("Prefix")
	log.W("Tag", "message")

	line := strings.TrimSpace(out.String())
	if !strings.Contains(line, "|W|") {
		t.Fatalf("missing level: %q", line)
	}
	if !strings.Contains(line, "|Prefix|Tag|") {
		t.Fatalf("missing prefix and tag: %q", line)
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
