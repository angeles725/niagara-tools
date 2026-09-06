# tests/lib/client-root.bash — the ONE blessed client read tree for every bats suite (C11 T2).
# Env override wins: ':=' only assigns when the caller left it unset/empty.
# Load at file scope (not inside setup()) with:   load lib/client-root
# [ev: design.md D3a; spec client-root-lib/spec.md R-T2.1..R-T2.3; apply-package 0ad09c658]
: "${CLIENT_READ_ROOT:=/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659}"
: "${C9_CLIENT_ROOT:=$CLIENT_READ_ROOT}"
: "${C9_CLIENT_REPO:=$CLIENT_READ_ROOT}"
: "${C8_CLIENT_REPO:=$CLIENT_READ_ROOT}"
export CLIENT_READ_ROOT C9_CLIENT_ROOT C9_CLIENT_REPO C8_CLIENT_REPO
