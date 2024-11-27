package retryMechanism

type RetryMechanism interface {
	Retry(operation func() error) error
}
