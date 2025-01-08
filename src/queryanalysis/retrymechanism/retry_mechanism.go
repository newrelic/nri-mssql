package retrymechanism

type RetryMechanism interface {
	Retry(operation func() error) error
}
