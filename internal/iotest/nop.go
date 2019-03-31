package iotest

type NopReader struct{}

func (NopReader) Read(buf []byte) (int, error) { return len(buf), nil }

type NopWriter struct{}

func (NopWriter) Write(buf []byte) (int, error) { return len(buf), nil }
