package configs

// ValidatingConfig is a config which can be validated.
type ValidatingConfig interface {
	Validate() error
}
