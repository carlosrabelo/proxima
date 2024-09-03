package script

import (
	"errors"
	"fmt"
	"strings"
)

// Params controls Alpine setup.sh generation.
type Params struct {
	RootPassword string
}

// Render returns the setup.sh body for the given parameters.
func Render(p Params) (string, error) {
	if p.RootPassword == "" {
		return "", errors.New("root_password is required")
	}

	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("set -e\n")
	b.WriteString("\n")
	b.WriteString("ALPINE_VERSION=\"$(cut -d. -f1,2 /etc/alpine-release)\"\n")
	b.WriteString("printf '%s\\n' \\\n")
	b.WriteString("  \"https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/main\" \\\n")
	b.WriteString("  \"https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/community\" \\\n")
	b.WriteString("  > /etc/apk/repositories\n")
	b.WriteString("\n")
	b.WriteString("apk update\n")
	b.WriteString("apk add --no-cache docker docker-compose qemu-guest-agent openssh\n")
	b.WriteString("\n")
	b.WriteString("rc-update add docker default\n")
	b.WriteString("rc-update add qemu-guest-agent default\n")
	b.WriteString("rc-update add sshd default\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("printf '%%s\\n' \"root:%s\" | chpasswd\n", p.RootPassword))
	b.WriteString("\n")
	b.WriteString("if grep -q '^PermitRootLogin' /etc/ssh/sshd_config; then\n")
	b.WriteString("  sed -i 's/^PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config\n")
	b.WriteString("else\n")
	b.WriteString("  printf '%s\\n' 'PermitRootLogin yes' >> /etc/ssh/sshd_config\n")
	b.WriteString("fi\n")

	return b.String(), nil
}
