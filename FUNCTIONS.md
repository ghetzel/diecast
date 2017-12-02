```go
asBool(value any) (bool, error)
```
Attempt to convert the given *value* to a boolean value.

```go
asFloat(value any) (float64, error)
```
Attempt to convert the given *value* to a floating-point number.

```go
asInt(value any) (int64, error)
```
Attempt to convert the given *value* to an integer.

```go
asStr(value any) (string, error)
```
Return the *value* as a string.

```go
asTime(value any) (Time, error)
```
Attempt to parse the given *value* as a date/time value.

```go
autobyte(bytesize any, ...any) (string, error)
```
Attempt to convert the given *bytesize* number to a string representation of the value in bytes.

```go
autotype(value any)
```
Attempt to automatically determine the type if *value* and return the converted output.

```go
basename(path any) string
```
Return the filename component of the given *path*.

```go
contains(s string, substr string) bool
```
Return whether a string *s* contains *substr*.

```go
dirname(path any) string
```
Return the directory path component of the given *path*.

```go
extname(path any) string
```
Return the extension component of the given *path* (always prefixed with a dot [.]).

```go
hasPrefix(s string, prefix string) bool
```
Return whether string *s* has the given *prefix*.

```go
hasSuffix(s string, suffix string) bool
```
Return whether string *s* has the given *suffix*.

```go
isArray(value any) bool
```
Return whether the given *value* is an iterable array or slice.

```go
isBool(value any) bool
```
Return whether the given *value* is a boolean type.

```go
isEmpty(value any) bool
```
Return whether the given *value* is empty.

```go
isFloat(value any) bool
```
Return whether the given *value* is a floating-point type.

```go
isInt(value any) bool
```
Return whether the given *value* is an integer type.

```go
isMap(value any) bool
```
Return whether the given *value* is a key-value map type.

```go
isZero(value any) bool
```
Return whether the given *value* is an zero-valued variable.

```go
join(input any, delimiter string) string
```
Join the *input* array on *delimiter* and return a string.

```go
jsonify(value any, indent ...any) (string, error)
```
Encode the given *value* as a JSON string, optionally using *indent* to pretty format the output.

```go
lower(s string) string
```
Return a copy of string *s* with all Unicode letters mapped to their lower case.

```go
ltrim(s string, prefix string) string
```
Return a copy of string *s* with the leading *prefix* removed.

```go
markdown(value any) (string, error)
```
Render the given Markdown string *value* as sanitized HTML.

```go
percent(value any, n ...any) (string, error)
```
Return the given floating point *value* as a percentage of *n*, or 100.0 if *n* is not specified.

```go
replace(s string, old string, new string, n int) string
```
Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.

```go
rtrim(s string, suffix string) string
```
Return a copy of string *s* with the trailing *suffix* removed.

```go
rxreplace(s any, pattern string, repl string) (string, error)
```
Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.

```go
split(s string, delimiter string, ...any)
```
Return a string array of elements resulting from *s* being split by *delimiter*, up to *n* times (if specified).

```go
strcount(s string, substr string) int
```
Count *s* for the number of non-overlapping instances of *substr*.

```go
surroundedBy(s any, prefix string, suffix string) bool
```
Return whether string *s* starts with *prefix* and ends with *suffix*.

```go
thousandify(value any, sep ...any) string
```
Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.

```go
titleize(s string) string
```
Return a copy of *s* with all Unicode letters that begin words mapped to their title case.

```go
trim(s string) string
```
Return a copy of *s* with all leading and trailing whitespace characters removed.

```go
upper(s string) string
```
Return a copy of *s* with all letters capitalized.

