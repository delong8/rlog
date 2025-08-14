package rlog

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
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
	inited       = false
	configed     = false
	rules        = []string{}
	ruleFile     = "./rlog"
	defaultRules = []string{"default"}
	// format the output data
	output func(v ...any) = log.Println
)

/*
Init is global function, give all instance default settings.
if no special configs, `nil` will be fine.
*/
func Init(cfg *Config) {
	// return if has been inited with not nil options
	if configed {
		return
	}
	// return if has been inited and current cfg is still nil
	if inited && cfg == nil {
		return
	}

	inited = true
	if cfg != nil {
		configed = true
	}

	interval := time.Duration(1 * time.Second)
	read := func() []string {
		rules, ok := defaultRead()
		if ok {
			return rules
		} else {
			return defaultRules
		}
	}

	if cfg != nil {
		if cfg.Interval > interval {
			interval = cfg.Interval
		}
		if cfg.Read != nil {
			read = cfg.Read
		}
		if cfg.Output != nil {
			output = cfg.Output
		}
		if cfg.File != "" {
			ruleFile = cfg.File
		}
		if cfg.DefaultRules != nil {
			defaultRules = cfg.DefaultRules
		}
	}

	// read init rules
	rules = read()

	go func() {
		for {
			time.Sleep(interval)
			rules = read()
		}
	}()
}

/*
return current rules config
*/
func Rules() []string {
	return rules
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
	output(prefix(true, 3, tag, v...)...)
}

func (l Logger) Info(v ...any) {
	Init(nil)
	if enabled(l.name) {
		output(prefix(false, 3, l.name, v...)...)
	}
}

/*
global error logger, without any configs, if you want give a error,
just use `rlog.Error()`
*/
func Error(v ...any) {
	output(prefix(true, 2, "[ERROR]", v...)...)
}

func Info(v ...any) {
	Init(nil)
	if !enabled("default") {
		return
	}
	output(prefix(false, 2, "INFO", v...)...)
}

func enabled(name string) bool {
	for _, i := range rules {
		if i == "*" {
			return true
		}
		if strings.HasPrefix(name, i) {
			return true
		}
	}
	return false
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
	f, err := os.Stat(ruleFile)

	if os.IsNotExist(err) || f.IsDir() || f.Size() > 1_000 {
		return []string{}, false
	}

	bs, err := os.ReadFile(ruleFile)
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
