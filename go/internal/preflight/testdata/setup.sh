# shellcheck shell=bash disable=SC2034  # SKIP is read by record.sh
# setup.sh — record.sh hooks of the preflight cases: <case>/.tools links the system tools TOOLS (as the Go
# test's linkTools does), so `#!/usr/bin/env bash` in a fake resolves through the case PATH alone.
TOOLS=(bash sh env grep sed awk cut find paste sort uniq wc tr comm basename dirname head tail xargs df sleep cat printf mkdir ls readlink timeout rm seq)
setup_case() {
  local t; mkdir -p "$D/.tools"
  for t in "${TOOLS[@]}"; do if [ -e "/usr/bin/$t" ]; then ln -s "/usr/bin/$t" "$D/.tools/$t"; elif [ -e "/bin/$t" ]; then ln -s "/bin/$t" "$D/.tools/$t"; fi; done
  SKIP=(.tools/)
}
