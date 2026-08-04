package redact

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestWriterRedactsSecretAcrossEveryWriteBoundary(t *testing.T) {
	const secret = "sentinel-registry-token"
	input := "before " + secret + " after"
	for split := 0; split <= len(input); split++ {
		t.Run(strings.Repeat("x", split), func(t *testing.T) {
			var output bytes.Buffer
			writer, err := NewWriter(&output, []string{secret})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write([]byte(input[:split])); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write([]byte(input[split:])); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if got, want := output.String(), "before [REDACTED] after"; got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
		})
	}
}

func TestWriterUsesLongestOverlappingSecretWithoutCorruptingOutput(t *testing.T) {
	var output bytes.Buffer
	writer, err := NewWriter(&output, []string{"token", "token-long", "long"})
	if err != nil {
		t.Fatal(err)
	}
	for _, chunk := range []string{"ordinary token", "-lo", "ng and longhouse"} {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "ordinary [REDACTED] and [REDACTED]house"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriterRejectsEmptySecretAndPreservesOrdinaryOutput(t *testing.T) {
	if _, err := NewWriter(&bytes.Buffer{}, []string{""}); err == nil {
		t.Fatal("empty secret was accepted")
	}
	var output bytes.Buffer
	writer, err := NewWriter(&output, []string{"sentinel"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("normal output")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "normal output" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestWriterSelectsReplacementThatCannotRecreateSecret(t *testing.T) {
	var output bytes.Buffer
	writer, err := NewWriter(&output, []string{"RED"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte("RED"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "RED") {
		t.Fatalf("replacement recreated secret: %q", output.String())
	}
}

func TestWriterSerializesConcurrentWrites(t *testing.T) {
	var output bytes.Buffer
	writer, err := NewWriter(&output, []string{"sentinel"})
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _ = writer.Write([]byte("ordinary\n"))
		}()
	}
	wait.Wait()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "ordinary\n") != 32 {
		t.Fatalf("output contains %d complete writes", strings.Count(output.String(), "ordinary\n"))
	}
}
