#!/bin/sh
set -e

ALPINE_VERSION="$(cut -d. -f1,2 /etc/alpine-release)"
printf '%s\n' \
  "https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/main" \
  "https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/community" \
  > /etc/apk/repositories

apk update
i=0
while [ "$i" -lt 3 ]; do
  if apk add --no-cache docker docker-compose qemu-guest-agent openssh; then
    break
  fi
  i=$((i + 1))
  if [ "$i" -ge 3 ]; then
    exit 1
  fi
  rm -rf /var/cache/apk/*
  apk update
  sleep 2
done

rc-update add docker default
rc-update add qemu-guest-agent default
rc-update add sshd default

printf '%s\n' "root:test-root-pass" | chpasswd

if grep -q '^PermitRootLogin' /etc/ssh/sshd_config; then
  sed -i 's/^PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
else
  printf '%s\n' 'PermitRootLogin yes' >> /etc/ssh/sshd_config
fi
