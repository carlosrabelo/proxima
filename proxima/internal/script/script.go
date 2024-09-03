package script

import "errors"

// Params controls Alpine setup.sh generation.
type Params struct {
	RootPassword string
}

// Render returns the setup.sh body for the given parameters.
func Render(p Params) (string, error) {
	if p.RootPassword == "" {
		return "", errors.New("root_password is required")
	}
	return "", nil
}
