package rcscan

// iterators are the array methods whose callback's first parameter is an element of the receiver.
var iterators = map[string]bool{"map": true, "forEach": true, "filter": true, "find": true, "findLast": true,
	"findIndex": true, "some": true, "every": true, "flatMap": true, "sort": true}

// literalMethods are the number and string methods whose result, called on literal text, is literal text too.
var literalMethods = map[string]bool{"toFixed": true, "toPrecision": true, "toString": true, "toUpperCase": true,
	"toLowerCase": true, "trim": true, "padStart": true, "padEnd": true}

// computedOps are the operators an expression computed from literals may use.
var computedOps = map[string]bool{"+": true, "-": true, "*": true, "/": true, "%": true, "**": true, "(": true,
	")": true, ",": true, "?": true, ":": true, "<": true, ">": true, "<=": true, ">=": true, "==": true, "!=": true,
	"===": true, "!==": true, "&&": true, "||": true, "!": true}

// writeOps are the operators that write the member chain before them.
var writeOps = map[string]bool{"=": true, "+=": true, "-=": true, "*=": true, "/=": true, "%=": true, "**=": true,
	"&&=": true, "||=": true, "??=": true, "<<=": true, ">>=": true, ">>>=": true, "&=": true, "|=": true, "^=": true,
	"++": true, "--": true}

// mutators are the methods that change the collection they are called on.
var mutators = map[string]bool{"push": true, "unshift": true, "splice": true, "fill": true, "copyWithin": true}
