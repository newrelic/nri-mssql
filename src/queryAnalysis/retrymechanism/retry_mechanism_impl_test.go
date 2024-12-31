package retrymechanism

import (
	"errors"
	"testing"
)

var (
	errOperationFailed = errors.New("operation failed")
	errTryAgain        = errors.New("try again")
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	retryMechanism := &RetryMechanismImpl{}

	err := retryMechanism.Retry(func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRetry_FailureAfterMaxRetries(t *testing.T) {
	retryMechanism := &RetryMechanismImpl{}
	attempts := 0

	err := retryMechanism.Retry(func() error {
		attempts++
		return errOperationFailed
	})

	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	if !errors.Is(err, errOperationFailed) {
		t.Fatalf("expected error %v, got %v", errOperationFailed, err)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_SuccessOnSubsequentAttempt(t *testing.T) {
	retryMechanism := &RetryMechanismImpl{}
	var attempts int

	err := retryMechanism.Retry(func() error {
		attempts++
		if attempts == 2 {
			return nil // Succeed on the second attempt
		}
		return errTryAgain
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
