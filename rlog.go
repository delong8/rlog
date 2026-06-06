package rlog

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	name string
}

type Config struct {
	File         string
	Read         func() []string
	Interval     time.Duration
	Output       func(v ...any)
	DefaultRules []string
}

var (
	mu             sync.RWMutex
	inited         = false
	configed       = false
	rules          = []string{}
	ruleFile       = "./rlog"
	defaultRules   = []string{"default"}
	readRules      func() []string
	pollingStarted = false
	// format the output data
	output func(v ...any) = log.Println
)

/*
Init is global function, give all instance default settings.
if no special configs, `nil` will be fine.
*/
func Init(cfg *Config) {
	interval := time.Duration(1 * time.Second)
	read := defaultReadRules
	startPolling := false

	mu.Lock()
	if configed {
		mu.Unlock()
		return
	}
	if inited && cfg == nil {
		mu.Unlock()
		return
	}

	inited = true
	if cfg != nil {
		configed = true
		if cfg.Interval > interval {
			interval = cfg.Interval
		}
		if cfg.Output != nil {
			output = cfg.Output
		}
		if cfg.File != "" {
			ruleFile = cfg.File
		}
		if cfg.DefaultRules != nil {
			defaultRules = copyStrings(cfg.DefaultRules)
		}
		if cfg.Read != nil {
			read = cfg.Read
		}
		readRules = read
	} else {
		if readRules == nil {
			readRules = defaultReadRules
		}
		read = readRules
	}
	if !pollingStarted {
		pollingStarted = true
		startPolling = true
	}
	mu.Unlock()

	setRules(read())

	if startPolling {
		go pollRules(interval)
	}
}

func pollRules(interval time.Duration) {
	for {
		time.Sleep(interval)
		read := getReadRules()
		if read == nil {
			continue
		}
		setRules(read())
	}
}

func defaultReadRules() []string {
	rules, ok := defaultRead()
	if ok {
		return rules
	}
	return getDefaultRules()
}

/*
return current rules config
*/
func Rules() []string {
	return getRules()
}

/*
create a new logger instance, every `Info` of this instance
will contain a named prefix with param of New
*/
func New(name string) Logger {
	return Logger{name: name}
}

/*
give a error log, contain a glaring red [ERROR] symbol,
and will always outprint in any log level
*/
func (l Logger) Error(v ...any) {
	tag := "[ERROR]"
	if l.name != "" {
		tag += " " + l.name
	}
	out := getOutput()
	out(prefix(true, 3, tag, v...)...)
}

func (l Logger) Info(v ...any) {
	Init(nil)
	if enabled(l.name) {
		out := getOutput()
		out(prefix(false, 3, l.name, v...)...)
	}
}

/*
global error logger, without any configs, if you want give a error,
just use `rlog.Error()`
*/
func Error(v ...any) {
	out := getOutput()
	out(prefix(true, 2, "[ERROR]", v...)...)
}

func Info(v ...any) {
	Init(nil)
	if !enabled("default") {
		return
	}
	out := getOutput()
	out(prefix(false, 2, "INFO", v...)...)
}

func enabled(name string) bool {
	for _, i := range getRules() {
		if i == "*" {
			return true
		}
		if strings.HasPrefix(name, i) {
			return true
		}
	}
	return false
}

func getRules() []string {
	mu.RLock()
	defer mu.RUnlock()

	return copyStrings(rules)
}

func setRules(next []string) {
	copied := copyStrings(next)

	mu.Lock()
	defer mu.Unlock()

	rules = copied
}

func getOutput() func(v ...any) {
	mu.RLock()
	defer mu.RUnlock()

	return output
}

func getReadRules() func() []string {
	mu.RLock()
	defer mu.RUnlock()

	return readRules
}

func getDefaultRules() []string {
	mu.RLock()
	defer mu.RUnlock()

	return copyStrings(defaultRules)
}

func getRuleFile() string {
	mu.RLock()
	defer mu.RUnlock()

	return ruleFile
}

func copyStrings(values []string) []string {
	return append([]string(nil), values...)
}

func prefix(err bool, skip int, tag string, v ...any) []any {
	_, file, line, ok := runtime.Caller(skip)

	if !ok {
		file = "???"
		line = 0
	}
	ps := strings.Split(file, "/")
	p := strings.Join(ps[len(ps)-2:], "/") + ":" + fmt.Sprintf("%d", line)

	t := []any{}
	if err {
		t = append(t, fmt.Sprintf("\x1b[91m%s\x1b[0m", tag))
	} else {
		if tag == "" {
			tag = "INFO"
		}
		t = append(t, fmt.Sprintf("[%s]", tag))
	}
	t = append(t, p)
	t = append(t, v...)
	return t
}

func defaultRead() ([]string, bool) {
	path := getRuleFile()
	f, err := os.Stat(path)

	if os.IsNotExist(err) {
		return []string{}, false
	}
	if err != nil {
		Error(err)
		return []string{}, false
	}
	if f.IsDir() || f.Size() > 1_000 {
		return []string{}, false
	}

	bs, err := os.ReadFile(path)
	if err != nil {
		Error(err)
		return []string{}, false
	}
	lines := strings.Split(strings.ReplaceAll(
		string(bs), "\r\n", "\n"), "\n")

	m := map[string]bool{}
	for _, line := range lines {
		// unique keys
		if line != "" {
			m[line] = true
		}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys, true
}
