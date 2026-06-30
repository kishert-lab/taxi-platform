package passenger

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestLoggingSMSServicePrintsCodeToConsole(t *testing.T) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	service := NewLoggingSMSService(zap.NewNop())
	if err := service.SendCode(context.Background(), "+79991234567", "1234"); err != nil {
		t.Fatalf("SendCode returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	var output bytes.Buffer
	if _, err := io.Copy(&output, reader); err != nil {
		t.Fatalf("read stdout output: %v", err)
	}

	consoleOutput := output.String()
	if !strings.Contains(consoleOutput, "PASSENGER_AUTH_CODE") {
		t.Fatalf("expected marker in output, got %q", consoleOutput)
	}
	if !strings.Contains(consoleOutput, "code=1234") {
		t.Fatalf("expected code in output, got %q", consoleOutput)
	}
}
