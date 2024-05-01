package ucl

import (
	"bytes"
	"io"
)

func LineHandler(line func(string)) io.Writer {
	return &lineHandlerWriter{
		lineBuffer: new(bytes.Buffer),
		writeLine:  line,
	}
}

type lineHandlerWriter struct {
	lineBuffer *bytes.Buffer
	writeLine  func(line string)
}

func (uo *lineHandlerWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if b == '\n' {
			uo.writeLine(uo.lineBuffer.String())
			uo.lineBuffer.Reset()
		} else {
			uo.lineBuffer.WriteByte(b)
		}
	}
	return len(p), nil
}
