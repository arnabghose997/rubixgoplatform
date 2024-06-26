package logger

import (
	"bytes"
	"io"
)

type writer struct {
	b     bytes.Buffer
	w     []io.Writer
	color []ColorOption
}

func newWriter(w []io.Writer, color []ColorOption) *writer {
	return &writer{w: w, color: color}
}

func (w *writer) Flush(level Level) (err error) {
	var unwritten = w.b.Bytes()

	for i, wr := range w.w {
		if lw, ok := wr.(LevelWriter); ok {
			_, err = lw.LevelWrite(level, unwritten)
		} else {
			l := len(unwritten)
			if unwritten[0] == 27 {
				unwritten = unwritten[5 : l-4]
			}
			if w.color[i] != ColorOff {
				color := _levelToColor[level]
				colorbytes := []byte(color.Sprintf("%s", unwritten))
				_, err = wr.Write(colorbytes)
			} else {
				_, err = wr.Write(unwritten)
			}

		}
	}

	w.b.Reset()
	return err
}

func (w *writer) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

func (w *writer) WriteByte(c byte) error {
	return w.b.WriteByte(c)
}

func (w *writer) WriteString(s string) (int, error) {
	return w.b.WriteString(s)
}

// LevelWriter is the interface that wraps the LevelWrite method.
type LevelWriter interface {
	LevelWrite(level Level, p []byte) (n int, err error)
}

// LeveledWriter writes all log messages to the standard writer,
// except for log levels that are defined in the overrides map.
type LeveledWriter struct {
	standard  io.Writer
	overrides map[Level]io.Writer
}

// NewLeveledWriter returns an initialized LeveledWriter.
//
// standard will be used as the default writer for all log levels,
// except for log levels that are defined in the overrides map.
func NewLeveledWriter(standard io.Writer, overrides map[Level]io.Writer) *LeveledWriter {
	return &LeveledWriter{
		standard:  standard,
		overrides: overrides,
	}
}

// Write implements io.Writer.
func (lw *LeveledWriter) Write(p []byte) (int, error) {
	return lw.standard.Write(p)
}

// LevelWrite implements LevelWriter.
func (lw *LeveledWriter) LevelWrite(level Level, p []byte) (int, error) {
	w, ok := lw.overrides[level]
	if !ok {
		w = lw.standard
	}
	return w.Write(p)
}
