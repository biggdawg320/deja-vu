package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Two aiders started together leave no header AiderSessionAt can call this
// run's: asker() caches only a non-empty answer, so once the header window
// (launch to launch+1m) has closed with nothing resolved, it still walks the
// whole history again on every later call instead of settling on the
// fallback sid (#5295).
func TestAiderAskerStopsWalkingOnceTheWindowCloses(t *testing.T) {
	tmp := t.TempDir()
	history := filepath.Join(tmp, ".aider.chat.history.md")

	launch, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-10-01 09:00:00", time.Local)
	if err != nil {
		t.Fatal(err)
	}
	// Bulk the file up so a full walk costs something measurable: what the
	// bug makes asker() pay again on every message for the rest of the run.
	padding := strings.Repeat(strings.Repeat("x", 200)+"\n", 100000)
	before := "\n# aider chat started at 2026-09-01 09:00:00\n\n#### filler  \n\nok\n\n"
	if err = os.WriteFile(history, []byte(padding+before), 0o644); err != nil {
		t.Fatal(err)
	}
	offset := int64(len(padding + before))
	// Two aiders started in the same launch second, both past offset: nothing
	// says which header is this run's.
	ambiguous := "\n# aider chat started at 2026-10-01 09:00:00\n\n#### one  \n\nok\n\n" +
		"\n# aider chat started at 2026-10-01 09:00:00\n\n#### two  \n\nok\n\n"
	f, err := os.OpenFile(history, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(ambiguous); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	l := &aiderLive{history: history, offset: offset, launch: launch, sid: "aider-test-fallback"}

	start := time.Now()
	first := l.asker()
	firstElapsed := time.Since(start)
	if first != l.sid {
		t.Fatalf("ambiguous launch: asker() returned %q, want the fallback %q", first, l.sid)
	}

	// The window is long closed: the answer is final and should be kept, not
	// walked for again.
	start = time.Now()
	second := l.asker()
	secondElapsed := time.Since(start)
	if second != l.sid {
		t.Fatalf("second call returned %q, want the same fallback %q", second, l.sid)
	}
	if secondElapsed > firstElapsed/4 {
		t.Fatalf("second call took %v, almost as long as the first (%v): asker() is still walking the whole history after the window closed", secondElapsed, firstElapsed)
	}
}
