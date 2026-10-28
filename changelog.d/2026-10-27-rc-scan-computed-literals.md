### Fixed — rc-scan innerhtml-server clears values computed from the page's own literals

- A field of a collection the page computes from its own const literal with `.map(u => ({ k: <expr> }))` (or a
  scalar `.map(u => <expr>)`), read directly, through a const arrow or through a function declaration with one
  return, is no `innerhtml-server` row when `<expr>` uses only literals, arithmetic and comparison operators,
  `Math.*`, members of literal elements, consts computed the same way, the callback's index and
  `toFixed`/`toString`-like methods. A `let`, an unbound name, a call, a template substitution, a loop-built
  array, a function with a second `return`, or a field the page writes into keeps the row.
