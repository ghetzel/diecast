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

<a name="asBool"></a>
```go
asBool(value any) (bool, error)
```
Attempt to convert the given *value* to a boolean value.



<a name="asFloat"></a>
```go
asFloat(value any) (float64, error)
```
Attempt to convert the given *value* to a floating-point number.



<a name="asInt"></a>
```go
asInt(value any) (int64, error)
```
Attempt to convert the given *value* to an integer.



<a name="asStr"></a>
```go
asStr(value any) (string, error)
```
Return the *value* as a string.



<a name="asTime"></a>
```go
asTime(value any) (Time, error)
```
Attempt to parse the given *value* as a date/time value.



<a name="autobyte"></a>
```go
autobyte(bytesize any, ...any) (string, error)
```
Attempt to convert the given *bytesize* number to a string representation of the value in bytes.



<a name="autotype"></a>
```go
autotype(value any)
```
Attempt to automatically determine the type if *value* and return the converted output.



<a name="basename"></a>
```go
basename(path any) string
```
Return the filename component of the given *path*.



<a name="contains"></a>
```go
contains(s string, substr string) bool
```
Return whether a string *s* contains *substr*.



<a name="dirname"></a>
```go
dirname(path any) string
```
Return the directory path component of the given *path*.



<a name="extname"></a>
```go
extname(path any) string
```
Return the extension component of the given *path* (always prefixed with a dot [.]).



<a name="hasPrefix"></a>
```go
hasPrefix(s string, prefix string) bool
```
Return whether string *s* has the given *prefix*.



<a name="hasSuffix"></a>
```go
hasSuffix(s string, suffix string) bool
```
Return whether string *s* has the given *suffix*.



<a name="isArray"></a>
```go
isArray(value any) bool
```
Return whether the given *value* is an iterable array or slice.



<a name="isBool"></a>
```go
isBool(value any) bool
```
Return whether the given *value* is a boolean type.



<a name="isEmpty"></a>
```go
isEmpty(value any) bool
```
Return whether the given *value* is empty.



<a name="isFloat"></a>
```go
isFloat(value any) bool
```
Return whether the given *value* is a floating-point type.



<a name="isInt"></a>
```go
isInt(value any) bool
```
Return whether the given *value* is an integer type.



<a name="isMap"></a>
```go
isMap(value any) bool
```
Return whether the given *value* is a key-value map type.



<a name="isZero"></a>
```go
isZero(value any) bool
```
Return whether the given *value* is an zero-valued variable.



<a name="join"></a>
```go
join(input any, delimiter string) string
```
Join the *input* array on *delimiter* and return a string.



<a name="jsonify"></a>
```go
jsonify(value any, indent ...any) (string, error)
```
Encode the given *value* as a JSON string, optionally using *indent* to pretty format the output.



<a name="lower"></a>
```go
lower(s string) string
```
Return a copy of string *s* with all Unicode letters mapped to their lower case.



<a name="ltrim"></a>
```go
ltrim(s string, prefix string) string
```
Return a copy of string *s* with the leading *prefix* removed.



<a name="markdown"></a>
```go
markdown(value any) (string, error)
```
Render the given Markdown string *value* as sanitized HTML.



<a name="percent"></a>
```go
percent(value any, n ...any) (string, error)
```
Return the given floating point *value* as a percentage of *n*, or 100.0 if *n* is not specified.



<a name="replace"></a>
```go
replace(s string, old string, new string, n int) string
```
Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.



<a name="rtrim"></a>
```go
rtrim(s string, suffix string) string
```
Return a copy of string *s* with the trailing *suffix* removed.



<a name="rxreplace"></a>
```go
rxreplace(s any, pattern string, repl string) (string, error)
```
Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.



<a name="split"></a>
```go
split(s string, delimiter string, ...any)
```
Return a string array of elements resulting from *s* being split by *delimiter*, up to *n* times (if specified).



<a name="strcount"></a>
```go
strcount(s string, substr string) int
```
Count *s* for the number of non-overlapping instances of *substr*.



<a name="surroundedBy"></a>
```go
surroundedBy(s any, prefix string, suffix string) bool
```
Return whether string *s* starts with *prefix* and ends with *suffix*.



<a name="thousandify"></a>
```go
thousandify(value any, sep ...any) string
```
Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.



<a name="titleize"></a>
```go
titleize(s string) string
```
Return a copy of *s* with all Unicode letters that begin words mapped to their title case.



<a name="trim"></a>
```go
trim(s string) string
```
Return a copy of *s* with all leading and trailing whitespace characters removed.



<a name="upper"></a>
```go
upper(s string) string
```
Return a copy of *s* with all letters capitalized.



