package message

import (
	"context"
	"errors"
	"testing"
)

func TestPollErrorIgnoresInternalDeadline(t *testing.T) {
	if err := pollError(context.DeadlineExceeded, context.DeadlineExceeded, nil); err != nil {
		t.Fatalf("pollError() = %v, want nil for the internal bounded poll timeout", err)
	}
}

func TestPollErrorPreservesKafkaAndRequestErrors(t *testing.T) {
	kafkaErr := errors.New("broker unavailable")
	if err := pollError(kafkaErr, nil, nil); !errors.Is(err, kafkaErr) {
		t.Fatalf("pollError() = %v, want Kafka error", err)
	}
	if err := pollError(context.DeadlineExceeded, context.DeadlineExceeded, context.Canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("pollError() = %v, want parent request error", err)
	}
}
