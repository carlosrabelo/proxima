package config

// Redact returns a copy of c with secrets replaced by "***" when non-empty.
func Redact(c Config) Config {
	out := c
	if out.Auth.Password != "" {
		out.Auth.Password = "***"
	}
	if out.Auth.TokenSecret != "" {
		out.Auth.TokenSecret = "***"
	}
	if out.RootPassword != "" {
		out.RootPassword = "***"
	}
	return out
}
