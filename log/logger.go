package log

import (
	"log"
)

const (
	LogLevel = LevelInfo
	Prefix   = "noise"
)

const (
	LevelInternal = iota
	LevelDebug
	LevelInfo
)

func Debug(message ...interface{}) {
	if LogLevel >= LevelDebug {
		log.Println(append([]interface{}{"[debug]"}, message...)...)
	}
}

func Print(message ...interface{}) {
	if LogLevel >= LevelInternal {
		log.Println(append([]interface{}{"[" + Prefix + "]"}, message...)...)
	}
}

func Info(message ...interface{}) {
	if LogLevel >= LevelInfo {
		log.Println(append([]interface{}{"[info]"}, message...)...)
	}
}
