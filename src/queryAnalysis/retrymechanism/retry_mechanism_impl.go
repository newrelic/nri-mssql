package retrymechanism

type RetryMechanismImpl struct{}

// Ensure SimpleRetry implements RetryMechanism
var _ RetryMechanism = (*RetryMechanismImpl)(nil)

func (r *RetryMechanismImpl) Retry(operation func() error) error {
	// Retry the operation 3 times(this we can add in some config file)
	var MaxRetries = 3
	var err error
	for i := 0; i < MaxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}
	}
	return err
}
