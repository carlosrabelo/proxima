package script

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Params controls Alpine setup.sh generation.
type Params struct {
	RootPassword     string
	SSHAuthorizedKey string // optional OpenSSH public key line for root
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
	// NBD-backed installs sometimes hit transient I/O errors on large packages
	// (e.g. docker-cli); retry after clearing the apk cache.
	b.WriteString("i=0\n")
	b.WriteString("while [ \"$i\" -lt 3 ]; do\n")
	b.WriteString("  if apk add --no-cache docker docker-compose qemu-guest-agent openssh; then\n")
	b.WriteString("    break\n")
	b.WriteString("  fi\n")
	b.WriteString("  i=$((i + 1))\n")
	b.WriteString("  if [ \"$i\" -ge 3 ]; then\n")
	b.WriteString("    exit 1\n")
	b.WriteString("  fi\n")
	b.WriteString("  rm -rf /var/cache/apk/*\n")
	b.WriteString("  apk update\n")
	b.WriteString("  sleep 2\n")
	b.WriteString("done\n")
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

	if key := normalizeAuthorizedKey(p.SSHAuthorizedKey); key != "" {
		b.WriteString("\n")
		b.WriteString("mkdir -p /root/.ssh\n")
		b.WriteString("chmod 700 /root/.ssh\n")
		b.WriteString(fmt.Sprintf("printf '%%s\\n' %s > /root/.ssh/authorized_keys\n", shellSingleQuote(key)))
		b.WriteString("chmod 600 /root/.ssh/authorized_keys\n")
	}

	return b.String(), nil
}

// WriteFile writes the rendered setup.sh to path with mode 0755.
func WriteFile(path string, p Params) error {
	content, err := Render(p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

func normalizeAuthorizedKey(raw string) string {
	key := strings.TrimSpace(raw)
	if key == "" {
		return ""
	}
	if i := strings.IndexByte(key, '\n'); i >= 0 {
		key = strings.TrimSpace(key[:i])
	}
	return key
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
