package kubernetes

import "io"

type LogPrefixWriter struct {
	prefix string
	writer io.Writer
}

func CreatePrefixWriter(prefix string, writer io.Writer) *LogPrefixWriter {
	return &LogPrefixWriter{prefix, writer}
}

func (pw *LogPrefixWriter) Write(p []byte) (n int, err error) {
	return pw.writer.Write(append([]byte(pw.prefix), p...))
}
