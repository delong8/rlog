package rlog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func resetForTest(t *testing.T) {
	t.Helper()
	resetGlobalsForTest()
	t.Cleanup(resetGlobalsForTest)
}

func resetGlobalsForTest() {
	mu.Lock()
	defer mu.Unlock()

	inited = false
	configed = false
	rules = []string{}
	ruleFile = "./rlog"
	defaultRules = []string{"default"}
	readRules = nil
	pollingStarted = false
	output = log.Println
}

type capturedOutput struct {
	mu    sync.Mutex
	calls [][]any
}

func (c *capturedOutput) fn(v ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	args := append([]any(nil), v...)
	c.calls = append(c.calls, args)
}

func (c *capturedOutput) strings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	lines := make([]string, 0, len(c.calls))
	for _, v := range c.calls {
		lines = append(lines, fmt.Sprint(v...))
	}
	return lines
}

func (c *capturedOutput) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.calls)
}

func TestResetForTestRestoresGlobals(t *testing.T) {
	resetForTest(t)

	mu.Lock()
	inited = true
	configed = true
	rules = []string{"mutated"}
	ruleFile = "mutated"
	defaultRules = []string{"mutated"}
	readRules = func() []string { return []string{"mutated"} }
	pollingStarted = true
	output = func(v ...any) {}
	mu.Unlock()

	resetForTest(t)

	mu.RLock()
	defer mu.RUnlock()
	if inited || configed {
		t.Fatalf("resetForTest left init flags set")
	}
	if len(rules) != 0 {
		t.Fatalf("resetForTest left rules = %v", rules)
	}
	if ruleFile != "./rlog" {
		t.Fatalf("resetForTest left ruleFile = %q", ruleFile)
	}
	if len(defaultRules) != 1 || defaultRules[0] != "default" {
		t.Fatalf("resetForTest left defaultRules = %v", defaultRules)
	}
	if readRules != nil || pollingStarted {
		t.Fatalf("resetForTest left readRules nil = %v, pollingStarted = %v", readRules == nil, pollingStarted)
	}
}

func TestCapturedOutputRecordsCalls(t *testing.T) {
	var capture capturedOutput

	capture.fn("a", 1)
	capture.fn("b")

	if got := capture.count(); got != 2 {
		t.Fatalf("capture.count() = %d, want 2", got)
	}
	got := capture.strings()
	if len(got) != 2 || got[0] != "a1" || got[1] != "b" {
		t.Fatalf("capture.strings() = %v", got)
	}
}

func requireRules(t *testing.T, want []string) {
	t.Helper()

	got := Rules()
	if len(got) != len(want) {
		t.Fatalf("Rules() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Rules() = %v, want %v", got, want)
		}
	}
}

func requireOutputContains(t *testing.T, lines []string, index int, substrings ...string) {
	t.Helper()

	if len(lines) <= index {
		t.Fatalf("captured output count = %d, want at least %d", len(lines), index+1)
	}
	for _, substring := range substrings {
		if !strings.Contains(lines[index], substring) {
			t.Fatalf("captured output[%d] = %q, want substring %q", index, lines[index], substring)
		}
	}
}

func sameStringsSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	seen := make(map[string]int, len(got))
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		if seen[value] == 0 {
			return false
		}
		seen[value]--
	}
	return true
}

func TestGlobalInfoUsesDefaultRulesAfterLazyInit(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Init(&Config{
		DefaultRules: []string{"default"},
		Output:       capture.fn,
		Interval:     time.Hour,
	})

	Info("visible")

	if got := capture.count(); got != 1 {
		t.Fatalf("capture.count() = %d, want 1", got)
	}
	requireOutputContains(t, capture.strings(), 0, "[INFO]", "visible")
}

func TestScopedInfoUsesPrefixRules(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Init(&Config{
		Read: func() []string {
			return []string{"scope"}
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})

	New("scope.child").Info("yes")
	New("other").Info("no")

	if got := capture.count(); got != 1 {
		t.Fatalf("capture.count() = %d, want 1", got)
	}
	requireOutputContains(t, capture.strings(), 0, "[scope.child]", "yes")
}

func TestErrorAlwaysOutputs(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Init(&Config{
		Read: func() []string {
			return []string{}
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})

	Error("err")
	New("blocked").Error("scoped")

	if got := capture.count(); got != 2 {
		t.Fatalf("capture.count() = %d, want 2", got)
	}
	lines := capture.strings()
	requireOutputContains(t, lines, 0, "[ERROR]", "err")
	requireOutputContains(t, lines, 1, "[ERROR]", "blocked", "scoped")
}

func TestNonNilInitCanFollowLazyNilInit(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Info("lazy")
	Init(&Config{
		Read: func() []string {
			return []string{"scope b"}
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})

	requireRules(t, []string{"scope b"})
	New("scope b").Info("configured")

	if got := capture.count(); got != 1 {
		t.Fatalf("capture.count() = %d, want 1", got)
	}
	requireOutputContains(t, capture.strings(), 0, "[scope b]", "configured")
}

func TestNonNilInitOnlyAppliesOnce(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Init(&Config{
		Read: func() []string {
			return []string{"scope b"}
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})
	Init(&Config{
		Read: func() []string {
			return []string{"scope c"}
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})

	requireRules(t, []string{"scope b"})
	New("scope b").Info("visible")
	New("scope c").Info("hidden")

	if got := capture.count(); got != 1 {
		t.Fatalf("capture.count() = %d, want 1", got)
	}
	requireOutputContains(t, capture.strings(), 0, "[scope b]", "visible")
}

func TestDefaultReadParsesRuleFile(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput
	path := filepath.Join(t.TempDir(), "rules")
	if err := os.WriteFile(path, []byte("a\r\nb\n\na\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	Init(&Config{
		File:     path,
		Output:   capture.fn,
		Interval: time.Hour,
	})

	if got, want := Rules(), []string{"a", "b"}; !sameStringsSet(got, want) {
		t.Fatalf("Rules() = %v, want set %v", got, want)
	}
}

func TestDefaultReadFallsBackForMissingDirectoryAndOversizedFiles(t *testing.T) {
	cases := []struct {
		name string
		path func(t *testing.T) string
	}{
		{
			name: "missing file",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing")
			},
		},
		{
			name: "directory",
			path: func(t *testing.T) string {
				return t.TempDir()
			},
		},
		{
			name: "oversized file",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "rules")
				if err := os.WriteFile(path, []byte(strings.Repeat("x", 1001)), 0o600); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
				return path
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetForTest(t)
			var capture capturedOutput

			Init(&Config{
				File:         tc.path(t),
				DefaultRules: []string{"fallback"},
				Output:       capture.fn,
				Interval:     time.Hour,
			})

			requireRules(t, []string{"fallback"})
		})
	}
}

func TestDefaultReadHandlesStatErrorsWithoutPanic(t *testing.T) {
	resetForTest(t)

	mu.Lock()
	ruleFile = string([]byte{'r', 0, 'l', 'o', 'g'})
	mu.Unlock()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("defaultRead() panicked: %v", recovered)
		}
	}()

	rules, ok := defaultRead()
	if ok {
		t.Fatalf("defaultRead() ok = true, want false")
	}
	if len(rules) != 0 {
		t.Fatalf("defaultRead() rules = %v, want empty", rules)
	}
}

func TestReadCallbackCanCallRulesWithoutDeadlock(t *testing.T) {
	resetForTest(t)

	done := make(chan struct{})
	go func() {
		Init(&Config{
			Read: func() []string {
				Rules()
				return []string{"x"}
			},
			Interval: time.Hour,
		})
		close(done)
	}()

	select {
	case <-done:
		requireRules(t, []string{"x"})
	case <-time.After(time.Second):
		t.Fatal("Init timed out; Read callback may have deadlocked calling Rules")
	}
}

func TestOutputCallbackCanCallRulesWithoutDeadlock(t *testing.T) {
	resetForTest(t)

	Init(&Config{
		Read: func() []string {
			return []string{"default"}
		},
		Output: func(v ...any) {
			Rules()
		},
		Interval: time.Hour,
	})

	done := make(chan struct{})
	go func() {
		Info("visible")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Info timed out; Output callback may have deadlocked calling Rules")
	}
}

func TestRulesReturnsDefensiveCopy(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Init(&Config{
		Read: func() []string {
			return []string{"a"}
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})

	got := Rules()
	got[0] = "*"

	requireRules(t, []string{"a"})
	New("blocked").Info("hidden")

	if got := capture.count(); got != 0 {
		t.Fatalf("capture.count() = %d, want 0", got)
	}
}

func TestDefaultRulesCopiedFromConfig(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput
	defaults := []string{"default"}

	Init(&Config{
		File:         filepath.Join(t.TempDir(), "missing-rules"),
		DefaultRules: defaults,
		Output:       capture.fn,
		Interval:     time.Hour,
	})
	defaults[0] = "*"

	requireRules(t, []string{"default"})
	Info("visible")
	New("blocked").Info("hidden")

	if got := capture.count(); got != 1 {
		t.Fatalf("capture.count() = %d, want 1", got)
	}
	requireOutputContains(t, capture.strings(), 0, "[INFO]", "visible")
}

func TestReadResultCopiedBeforeStore(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput
	returned := []string{"a"}

	Init(&Config{
		Read: func() []string {
			return returned
		},
		Output:   capture.fn,
		Interval: time.Hour,
	})
	returned[0] = "*"

	requireRules(t, []string{"a"})
	New("blocked").Info("hidden")

	if got := capture.count(); got != 0 {
		t.Fatalf("capture.count() = %d, want 0", got)
	}
}

func TestConcurrentPollingAndRulesReadsAreRaceFree(t *testing.T) {
	resetForTest(t)

	var capture capturedOutput
	reads := make(chan []string, 16)
	reads <- []string{"default"}
	reads <- []string{"scope"}
	Init(&Config{
		Read: func() []string {
			select {
			case next := <-reads:
				return next
			default:
				return []string{"default"}
			}
		},
		Output:   capture.fn,
		Interval: time.Millisecond,
	})

	deadline := time.After(25 * time.Millisecond)
	for {
		select {
		case <-deadline:
			return
		default:
			Rules()
			Info("visible")
			New("scope.child").Info("scoped")
		}
	}
}

func TestLazyInitPollingUsesLaterConfiguredReadProvider(t *testing.T) {
	resetForTest(t)
	var capture capturedOutput

	Info("lazy")
	Init(&Config{
		Read: func() []string {
			return []string{"configured"}
		},
		Output:   capture.fn,
		Interval: time.Millisecond,
	})

	requireRules(t, []string{"configured"})
	New("configured.child").Info("visible")

	if got := capture.count(); got != 1 {
		t.Fatalf("capture.count() = %d, want 1", got)
	}
	requireOutputContains(t, capture.strings(), 0, "[configured.child]", "visible")
}
