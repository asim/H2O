package healthcheck

// Checker is how healthchecks implement the logic for testing health and returning samples
type Checker func() (measurements map[string]string, err error)
