package write

import (
	"bytes"
	"testing"
)

func TestArchiveOutputNeverExceedsCeiling(t *testing.T) {
	var data bytes.Buffer
	output := &backupOutput{writer: &data, remaining: 4}
	if n, err := output.Write([]byte("123")); n != 3 || err != nil {
		t.Fatalf("first write: %d %v", n, err)
	}
	n, err := output.Write([]byte("45"))
	errorCode(t, err, "INPUT_TOO_LARGE")
	if n != 0 || data.String() != "123" {
		t.Fatalf("oversized write leaked bytes: %d %q", n, data.String())
	}
	if n, err := output.Write([]byte("4")); n != 1 || err != nil || data.String() != "1234" {
		t.Fatalf("exact boundary write: %d %v %q", n, err, data.String())
	}
}
