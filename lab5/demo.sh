#!/usr/bin/env bash
# Demo runner for go2web. Drives the binary through every scoring item so
# the run can be screen-recorded into assets/demo.gif.

set -e
cd "$(dirname "$0")"

if [ ! -x ./go2web ]; then
  echo "→ building go2web"
  make build
fi

# fresh cache so demo shows miss → store → hit
rm -rf "$HOME/.go2web"

step() {
  printf "\n\033[1;36m$ %s\033[0m\n" "$*"
  sleep 1
  eval "$*"
  sleep 2
}

step "./go2web -h"

step "./go2web -u https://example.com"

step "./go2web -u https://api.github.com/users/octocat | head -20"

step "./go2web -v -u http://github.com 2>&1 | head -8"

step "./go2web -s coffee shop chisinau"

step "./go2web -v -u https://api.github.com/users/octocat 2>&1 | grep -E 'cache|GET' | head -5"
echo "  ↑ first call: cache miss → store"
sleep 2

step "./go2web -v -u https://api.github.com/users/octocat 2>&1 | grep cache"
echo "  ↑ second call: cache hit, no network"

echo
echo "demo done."
