# cases.sh — the golden cases, sourced by record.sh: bats-* mirror tests/generate-wiring-map.bats runs, edge-* are
# extra. new: a fresh case dir $D; fx: copy the bats fixtures in; put <path> <body>: write a file (printf %b);
# EXP: files the run must leave; rec <name> <args...>: run and write <name>.txt.
# shellcheck shell=bash disable=SC2034  # EXP is read by record.sh
fx() { cp -r "$FIX/generate-wiring-map" "$D/fx"; cp -r "$FIX/generate-wiring-map-lex" "$D/fx-lex"; }
op() { printf '@NiagaraProperty(name = "%s", type = "double", defaultValue = "0", flags = Flags.%s)\\n' "$1" "${2:-OPERATOR}"; }

# --- tests/generate-wiring-map.bats ---
new; rec bats-usage
new; rec bats-notdir /nonexistent/__gwm_xyz__
new; fx; rec bats-unknown-opt "$D/fx" --unknown-flag
new; fx; rec bats-tables "$D/fx"
new; fx; EXP=(wiring-map.md); rec bats-output-file "$D/fx" --output "$D/wiring-map.md"
new; mkdir -p "$D/empty_gwm"; rec bats-empty-src "$D/empty_gwm"
new; fx; rec bats-display "$D/fx-lex/Facade-rt/src" --lexicon "$D/fx-lex/Facade-rt/module.lexicon"
new; fx; rec bats-display-auto "$D/fx-lex/Facade-rt/src"
new; fx; rec bats-lexicon-missing "$D/fx" --lexicon /nonexistent/__gwm__.lexicon
new; put src/A.java 'class A {}\n'; mkdir -p "$D/src/locked"; chmod 000 "$D/src/locked"; rec bats-locked "$D/src"

# --- usage and options ---
new; fx; rec edge-output-no-arg "$D/fx" --output
new; fx; rec edge-lexicon-no-arg "$D/fx" --lexicon
new; fx; rec edge-output-dash "$D/fx" --output -
new; fx; rec edge-option-first --output "$D/o.md"
new; fx; rec edge-extra-positional "$D/fx" "$D/fx"
new; fx; rec edge-lexicon-before-unknown "$D/fx" --lexicon /nonexistent/x.lexicon --bogus
new; fx; rec edge-unknown-before-lexicon "$D/fx" --bogus --lexicon /nonexistent/x.lexicon
new; fx; mkdir -p "$D/lexdir"; rec edge-lexicon-is-dir "$D/fx" --lexicon "$D/lexdir"
new; fx; put a.lexicon 'temperatureSetpoint=A\n'; put b.lexicon 'temperatureSetpoint=B\n'
rec edge-lexicon-last-wins "$D/fx" --lexicon "$D/a.lexicon" --lexicon "$D/b.lexicon"
new; fx; EXP=(b.md); rec edge-output-last-wins "$D/fx" --output "$D/a.md" --output "$D/b.md"
new; put f.java "$(op x)"; rec edge-src-is-file "$D/f.java"
new; rec edge-src-empty-string ''
new; fx; rec edge-src-relative fx
new; fx; rec edge-src-trailing-slash "$D/fx/"
new; fx; rec edge-src-dot-relative ./fx/.
new; put 'my src/BA.java' "$(op a)"; rec edge-src-space "$D/my src"
new; put real/BA.java "$(op a)"; ln -s real "$D/link"; rec edge-src-symlink-root "$D/link"
new; fx; EXP=(out.md); rec edge-output-relative "$D/fx" --output out.md

# --- lexicon discovery and reading ---
new; put m/src/BA.java "$(op a)"; put m/src/module.lexicon 'a=In src\n'; put m/module.lexicon 'a=In parent\n'
rec edge-lexicon-src-wins "$D/m/src"
new; put m/src/BA.java "$(op a)"; put m/module.lexicon 'a=In parent\n'; put m/x.lexicon 'a=Explicit\n'
rec edge-lexicon-explicit-wins "$D/m/src" --lexicon "$D/m/x.lexicon"
new; put m/src/BA.java "$(op a)"; mkdir -p "$D/m/src/module.lexicon"; put m/module.lexicon 'a=Parent\n'
rec edge-lexicon-src-dir-skipped "$D/m/src"
new; put src/BA.java "$(op a)$(op b)$(op c)$(op d)$(op e)$(op f)$(op g)$(op h)"
put src/module.lexicon '# a=comment\n  # b=indented comment\nno equals line\n  c  =  spaced value  \nd=first\nd=second\ne=a|b|c\nf=\ng=x=y\r\nh=\\u00e9 back\\\\slash\n=nokey\n'
rec edge-lexicon-shapes "$D/src"
new; put src/BA.java "$(op 'a b')$(op t)"; put src/module.lexicon 'a b=spaced key\nt=tab\there\n'
rec edge-lexicon-inner-blank "$D/src"
new; put src/BA.java "$(op a)"; put src/module.lexicon 'a=Caf\xc3\xa9 \xe9\n'; rec edge-lexicon-bytes "$D/src"
new; put src/BA.java "$(op a)"; put src/module.lexicon 'a=no newline'; rec edge-lexicon-noeol "$D/src"

# --- slot harvest ---
new; put src/BA.java '@NiagaraProperty(\n  name = "multi",\n  type = "double",\n  flags = Flags.OPERATOR\n)\n'
rec edge-multiline "$D/src"
new; put src/BA.java "$(op a)$(op b SUMMARY)$(op c READONLY)$(op d HIDDEN)$(op e 'SUMMARY | Flags.OPERATOR')"
rec edge-flags "$D/src"
new; put src/BA.java "$(op a)\n$(op b SUMMARY)"; rec edge-blank-between "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "one", flags = Flags.OPERATOR) @NiagaraProperty(name = "two", flags = Flags.SUMMARY)\n'
rec edge-two-on-one-line "$D/src"
new; put src/BA.java '@NiagaraProperty(type = "double", flags = Flags.OPERATOR)\n@NiagaraProperty(name="", flags = Flags.SUMMARY)\n'
rec edge-no-name "$D/src"
new; put src/BA.java '@NiagaraProperty(name="tight",flags=Flags.OPERATOR)\n@NiagaraProperty(name\t=\t"tabbed", flags = Flags.SUMMARY)\n'
rec edge-name-spacing "$D/src"
new; put src/BA.java '@NiagaraProperty(displayname = "first", name = "second", flags = Flags.OPERATOR)\n'
rec edge-name-suffix "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "nested", defaultValue = "BDouble.make(Math.max(1, 2))", flags = Flags.OPERATOR)\n'
rec edge-nested-parens "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "open", flags = Flags.OPERATOR,\n  defaultValue = "x"\n'
rec edge-unbalanced-eof "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "lit", defaultValue = "BString.make(\\"(\\")", flags = Flags.OPERATOR)\n'"$(op after SUMMARY)"
rec edge-paren-in-literal "$D/src"
new; put src/BA.java '// a note on @NiagaraProperty usage\n'"$(op real)$(op next SUMMARY)"
rec edge-comment-no-paren "$D/src"
# edge-commented-out and edge-javadoc-open-paren became truth-commented-out and truth-javadoc-open-paren
# (the scan reads code, not comments); they are not re-recorded.
new; put src/BA.java '@NiagaraProperty\n(name = "nextline", flags = Flags.OPERATOR)\n'; rec edge-paren-next-line "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "crlf", flags = Flags.OPERATOR)\r\n'; rec edge-crlf "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "noeol", flags = Flags.OPERATOR)'; rec edge-java-noeol "$D/src"
new; put src/BA.java '@NiagaraProperties({\n  @NiagaraProperty(name = "w1", flags = Flags.OPERATOR),\n  @NiagaraProperty(name = "w2", flags = Flags.SUMMARY)\n})\n'
rec edge-properties-wrapper "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "x\ty", flags = Flags.OPERATOR)\n@NiagaraProperty(name = "\tlead", flags = Flags.SUMMARY)\n'; put src/module.lexicon 'x=X\n'
rec edge-tab-in-name "$D/src"
new; put src/BA.java '@NiagaraProperty(name = "OPERATORish", flags = Flags.SUMMARY)\n'; rec edge-operator-in-name "$D/src"

# --- walk ---
new; put src/b/BZ.java "$(op z)"; put src/BA.java "$(op a)"; put src/a.java "$(op lower)"; put src/B_.java "$(op under)"
rec edge-order "$D/src"
new; put src/BA.java "$(op a)"; put src/.hidden/BH.java "$(op hidden)"; put src/x-worktrees/BW.java "$(op wt)"
put src/.BD.java "$(op dotfile)"; put src/BC.JAVA "$(op upper)"; put src/BT.java.txt "$(op txt)"; put src/build/BB.java "$(op build)"
rec edge-prune "$D/src"
new; put src/BA.java "$(op a)"; put other/BO.java "$(op linked)"; ln -s ../other/BO.java "$D/src/BL.java"; ln -s ../other "$D/src/dirlink"
rec edge-symlinks-below "$D/src"
new; put src/x/BA.java "$(op a)"; put src/y/BA.java "$(op a SUMMARY)"; put src/module.lexicon 'a=Same\n'
rec edge-duplicate-slot "$D/src"
new; put src/BA.java "$(op a)"; put src/BU.java "$(op u)"; chmod 000 "$D/src/BU.java"; rec edge-unreadable-java "$D/src"
new; put src/BA.java "$(op a)"; put src/module.lexicon 'a=X\n'; chmod 000 "$D/src/module.lexicon"; rec edge-unreadable-lexicon "$D/src"
new; put src/A.java 'class A {}\n'; mkdir -p "$D/src/z/locked" "$D/src/a/locked"; chmod 000 "$D/src/z/locked" "$D/src/a/locked"
rec edge-two-locked "$D/src"
new; put src/BRoom.java "$(op a)"; put src/Bx.java "$(op b)"; put src/B.java "$(op c)"; rec edge-type-names "$D/src"
