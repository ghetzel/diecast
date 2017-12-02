# Diecast Function Reference

Diecast templates have access to a standard set of functions that aim to make
working with data and building web pages easier. Use the reference below to see
which functions are available and how to use them.

## Function List

- [asBool](#asBool)
- [asFloat](#asFloat)
- [asInt](#asInt)
- [asStr](#asStr)
- [asTime](#asTime)
- [autobyte](#autobyte)
- [autotype](#autotype)
- [basename](#basename)
- [contains](#contains)
- [dirname](#dirname)
- [extname](#extname)
- [hasPrefix](#hasPrefix)
- [hasSuffix](#hasSuffix)
- [isArray](#isArray)
- [isBool](#isBool)
- [isEmpty](#isEmpty)
- [isFloat](#isFloat)
- [isInt](#isInt)
- [isMap](#isMap)
- [isZero](#isZero)
- [join](#join)
- [jsonify](#jsonify)
- [lower](#lower)
- [ltrim](#ltrim)
- [markdown](#markdown)
- [percent](#percent)
- [replace](#replace)
- [rtrim](#rtrim)
- [rxreplace](#rxreplace)
- [split](#split)
- [strcount](#strcount)
- [surroundedBy](#surroundedBy)
- [thousandify](#thousandify)
- [titleize](#titleize)
- [trim](#trim)
- [upper](#upper)
## Function Usage

<hr />
<a name="asBool"></a>
```go
asBool(value any) (bool, error)
```
Attempt to convert the given *value* to a boolean value.

<hr />
<a name="asFloat"></a>
```go
asFloat(value any) (float64, error)
```
Attempt to convert the given *value* to a floating-point number.

<hr />
<a name="asInt"></a>
```go
asInt(value any) (int64, error)
```
Attempt to convert the given *value* to an integer.

<hr />
<a name="asStr"></a>
```go
asStr(value any) (string, error)
```
Return the *value* as a string.

<hr />
<a name="asTime"></a>
```go
asTime(value any) (Time, error)
```
Attempt to parse the given *value* as a date/time value.

<hr />
<a name="autobyte"></a>
```go
autobyte(bytesize any, ...any) (string, error)
```
Attempt to convert the given *bytesize* number to a string representation of the value in bytes.

<hr />
<a name="autotype"></a>
```go
autotype(value any)
```
Attempt to automatically determine the type if *value* and return the converted output.

<hr />
<a name="basename"></a>
```go
basename(path any) string
```
Return the filename component of the given *path*.

<hr />
<a name="contains"></a>
```go
contains(s string, substr string) bool
```
Return whether a string *s* contains *substr*.

<hr />
<a name="dirname"></a>
```go
dirname(path any) string
```
Return the directory path component of the given *path*.

<hr />
<a name="extname"></a>
```go
extname(path any) string
```
Return the extension component of the given *path* (always prefixed with a dot [.]).

<hr />
<a name="hasPrefix"></a>
```go
hasPrefix(s string, prefix string) bool
```
Return whether string *s* has the given *prefix*.

<hr />
<a name="hasSuffix"></a>
```go
hasSuffix(s string, suffix string) bool
```
Return whether string *s* has the given *suffix*.

<hr />
<a name="isArray"></a>
```go
isArray(value any) bool
```
Return whether the given *value* is an iterable array or slice.

<hr />
<a name="isBool"></a>
```go
isBool(value any) bool
```
Return whether the given *value* is a boolean type.

<hr />
<a name="isEmpty"></a>
```go
isEmpty(value any) bool
```
Return whether the given *value* is empty.

<hr />
<a name="isFloat"></a>
```go
isFloat(value any) bool
```
Return whether the given *value* is a floating-point type.

<hr />
<a name="isInt"></a>
```go
isInt(value any) bool
```
Return whether the given *value* is an integer type.

<hr />
<a name="isMap"></a>
```go
isMap(value any) bool
```
Return whether the given *value* is a key-value map type.

<hr />
<a name="isZero"></a>
```go
isZero(value any) bool
```
Return whether the given *value* is an zero-valued variable.

<hr />
<a name="join"></a>
```go
join(input any, delimiter string) string
```
Join the *input* array on *delimiter* and return a string.

<hr />
<a name="jsonify"></a>
```go
jsonify(value any, indent ...any) (string, error)
```
Encode the given *value* as a JSON string, optionally using *indent* to pretty format the output.

<hr />
<a name="lower"></a>
```go
lower(s string) string
```
Return a copy of string *s* with all Unicode letters mapped to their lower case.

<hr />
<a name="ltrim"></a>
```go
ltrim(s string, prefix string) string
```
Return a copy of string *s* with the leading *prefix* removed.

<hr />
<a name="markdown"></a>
```go
markdown(value any) (string, error)
```
Render the given Markdown string *value* as sanitized HTML.

<hr />
<a name="percent"></a>
```go
percent(value any, n ...any) (string, error)
```
Return the given floating point *value* as a percentage of *n*, or 100.0 if *n* is not specified.

<hr />
<a name="replace"></a>
```go
replace(s string, old string, new string, n int) string
```
Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.

<hr />
<a name="rtrim"></a>
```go
rtrim(s string, suffix string) string
```
Return a copy of string *s* with the trailing *suffix* removed.

<hr />
<a name="rxreplace"></a>
```go
rxreplace(s any, pattern string, repl string) (string, error)
```
Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.

<hr />
<a name="split"></a>
```go
split(s string, delimiter string, ...any)
```
Return a string array of elements resulting from *s* being split by *delimiter*, up to *n* times (if specified).

<hr />
<a name="strcount"></a>
```go
strcount(s string, substr string) int
```
Count *s* for the number of non-overlapping instances of *substr*.

<hr />
<a name="surroundedBy"></a>
```go
surroundedBy(s any, prefix string, suffix string) bool
```
Return whether string *s* starts with *prefix* and ends with *suffix*.

<hr />
<a name="thousandify"></a>
```go
thousandify(value any, sep ...any) string
```
Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.

<hr />
<a name="titleize"></a>
```go
titleize(s string) string
```
Return a copy of *s* with all Unicode letters that begin words mapped to their title case.

<hr />
<a name="trim"></a>
```go
trim(s string) string
```
Return a copy of *s* with all leading and trailing whitespace characters removed.

<hr />
<a name="upper"></a>
```go
upper(s string) string
```
Return a copy of *s* with all letters capitalized.

