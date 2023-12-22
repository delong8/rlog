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
	Print        func(v ...any)
	DefaultRules []string
}

var (
	inited                  = false
	configed                = false
	rules                   = []string{"default"}
	ruleFile                = "./rlog"
	print    func(v ...any) = log.Println
)

// 初始化时一般直接配置 nil 既可
func Init(cfg *Config) {
	if configed {
		return
	}
	if inited && cfg == nil {
		return
	}

	inited = true
	if cfg != nil {
		configed = true
	}

	interval := time.Duration(1 * time.Second)
	read := func() []string {
		f, err := os.Stat(ruleFile)

		if os.IsNotExist(err) || f.IsDir() || f.Size() > 10_000 {
			return []string{}
		}

		bs, err := os.ReadFile(ruleFile)
		if err != nil {
			Error(err)
		}
		lines := strings.Split(strings.ReplaceAll(
			string(bs), "\r\n", "\n"), "\n")

		m := map[string]bool{}
		for _, line := range lines {
			if line != "" {
				m[line] = true
			}
		}

		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}

		return keys
	}

	if cfg != nil {
		if cfg.Interval > interval {
			interval = cfg.Interval
		}
		if cfg.Read != nil {
			read = cfg.Read
		}
		if cfg.Print != nil {
			print = cfg.Print
		}
		if cfg.File != "" {
			ruleFile = cfg.File
		}
		if cfg.DefaultRules != nil {
			rules = cfg.DefaultRules
		}
	}

	go func() {
		for {
			time.Sleep(interval)
			rules = read()
		}
	}()
}

func New(name string) Logger {
	return Logger{name: name}
}

func (l Logger) Error(v ...any) {
	tag := "[ERROR]"
	if l.name != "" {
		tag += " " + l.name
	}
	print(prefix(true, 3, tag, v...)...)
}

func (l Logger) Info(v ...any) {
	Init(nil)
	if enabled(l.name) {
		print(prefix(false, 3, l.name, v...)...)
	}
}

func Error(v ...any) {
	print(prefix(true, 2, "[ERROR]", v...)...)
}

func Info(v ...any) {
	Init(nil)
	if !enabled("default") {
		return
	}
	print(prefix(false, 2, "INFO", v...)...)
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
