package redact

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"sync"
)

var replacementCandidates = [][]byte{
	[]byte("[REDACTED]"),
	[]byte("***"),
	[]byte("[HIDDEN]"),
}

type Writer struct {
	mu          sync.Mutex
	destination io.Writer
	secrets     [][]byte
	replacement []byte
	pending     []byte
	closed      bool
	err         error
}

func NewWriter(destination io.Writer, values []string) (*Writer, error) {
	if destination == nil {
		return nil, errors.New("redaction destination is required")
	}
	unique := make(map[string]struct{}, len(values))
	secrets := make([][]byte, 0, len(values))
	for _, value := range values {
		if value == "" {
			return nil, errors.New("secret value must not be empty")
		}
		if _, exists := unique[value]; exists {
			continue
		}
		unique[value] = struct{}{}
		secrets = append(secrets, []byte(value))
	}
	sort.Slice(secrets, func(i, j int) bool {
		if len(secrets[i]) == len(secrets[j]) {
			return bytes.Compare(secrets[i], secrets[j]) < 0
		}
		return len(secrets[i]) > len(secrets[j])
	})
	return &Writer{destination: destination, secrets: secrets, replacement: safeReplacement(secrets)}, nil
}

func (w *Writer) Write(contents []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, errors.New("write to closed redactor")
	}
	if w.err != nil {
		return 0, w.err
	}
	if len(w.secrets) == 0 {
		written, err := w.destination.Write(contents)
		if err != nil {
			w.err = err
		}
		return written, err
	}
	w.pending = append(w.pending, contents...)
	w.process(false)
	return len(contents), w.err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return w.err
	}
	w.closed = true
	if w.err == nil {
		w.process(true)
	}
	return w.err
}

func (w *Writer) process(final bool) {
	for len(w.pending) > 0 && w.err == nil {
		exactLength, possibleLonger := w.matchPrefix()
		switch {
		case exactLength > 0 && (final || !possibleLonger):
			w.write(w.replacement)
			w.pending = w.pending[exactLength:]
		case possibleLonger && !final:
			return
		default:
			w.write(w.pending[:1])
			w.pending = w.pending[1:]
		}
	}
}

func (w *Writer) matchPrefix() (exactLength int, possibleLonger bool) {
	for _, secret := range w.secrets {
		switch {
		case len(w.pending) >= len(secret) && bytes.Equal(w.pending[:len(secret)], secret):
			if len(secret) > exactLength {
				exactLength = len(secret)
			}
		case len(w.pending) < len(secret) && bytes.Equal(w.pending, secret[:len(w.pending)]):
			possibleLonger = true
		}
	}
	return exactLength, possibleLonger
}

func (w *Writer) write(contents []byte) {
	if len(contents) == 0 {
		return
	}
	written, err := w.destination.Write(contents)
	if err != nil {
		w.err = err
	} else if written != len(contents) {
		w.err = io.ErrShortWrite
	}
}

func safeReplacement(secrets [][]byte) []byte {
	for _, candidate := range replacementCandidates {
		safe := true
		for _, secret := range secrets {
			if bytes.Contains(candidate, secret) {
				safe = false
				break
			}
		}
		if safe {
			return candidate
		}
	}
	return nil
}
