package rlog

import (
	"testing"
	"time"
)

func Test_basic(t *testing.T) {
	Error("basic error")
	Info("basic info1")
	time.Sleep(2 * time.Second)
	Info("basic info2")
	Info(Rules())
}

func Test_logger(t *testing.T) {
	a := New("scope a")
	a.Error("error a")
	a.Info("info a")
}

func Test_confg(t *testing.T) {
	Init(&Config{
		Read: func() []string {
			return []string{"scope b"}
		},
	})

	b := New("scope b")

	Error("error default")
	Info("info default")
	b.Error("error b")
	b.Info("info b")

	Init(&Config{
		Read: func() []string {
			return []string{"scope b"}
		},
	})

	time.Sleep(2 * time.Second)
}
