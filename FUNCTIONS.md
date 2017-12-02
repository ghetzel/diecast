# Diecast Function Reference

Diecast templates have access to a standard set of functions that aim to make
working with data and building web pages easier. Use the reference below to see
which functions are available and how to use them.

## Functions

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
<a name="asBool">
```go
asBool(value any) (bool, error)
```
Attempt to convert the given *value* to a boolean value.

<a name="asFloat">
```go
asFloat(value any) (float64, error)
```
Attempt to convert the given *value* to a floating-point number.

<a name="asInt">
```go
asInt(value any) (int64, error)
```
Attempt to convert the given *value* to an integer.

<a name="asStr">
```go
asStr(value any) (string, error)
```
Return the *value* as a string.

<a name="asTime">
```go
asTime(value any) (Time, error)
```
Attempt to parse the given *value* as a date/time value.

<a name="autobyte">
```go
autobyte(bytesize any, ...any) (string, error)
```
Attempt to convert the given *bytesize* number to a string representation of the value in bytes.

<a name="autotype">
```go
autotype(value any)
```
Attempt to automatically determine the type if *value* and return the converted output.

<a name="basename">
```go
basename(path any) string
```
Return the filename component of the given *path*.

<a name="contains">
```go
contains(s string, substr string) bool
```
Return whether a string *s* contains *substr*.

<a name="dirname">
```go
dirname(path any) string
```
Return the directory path component of the given *path*.

<a name="extname">
```go
extname(path any) string
```
Return the extension component of the given *path* (always prefixed with a dot [.]).

<a name="hasPrefix">
```go
hasPrefix(s string, prefix string) bool
```
Return whether string *s* has the given *prefix*.

<a name="hasSuffix">
```go
hasSuffix(s string, suffix string) bool
```
Return whether string *s* has the given *suffix*.

<a name="isArray">
```go
isArray(value any) bool
```
Return whether the given *value* is an iterable array or slice.

<a name="isBool">
```go
isBool(value any) bool
```
Return whether the given *value* is a boolean type.

<a name="isEmpty">
```go
isEmpty(value any) bool
```
Return whether the given *value* is empty.

<a name="isFloat">
```go
isFloat(value any) bool
```
Return whether the given *value* is a floating-point type.

<a name="isInt">
```go
isInt(value any) bool
```
Return whether the given *value* is an integer type.

<a name="isMap">
```go
isMap(value any) bool
```
Return whether the given *value* is a key-value map type.

<a name="isZero">
```go
isZero(value any) bool
```
Return whether the given *value* is an zero-valued variable.

<a name="join">
```go
join(input any, delimiter string) string
```
Join the *input* array on *delimiter* and return a string.

<a name="jsonify">
```go
jsonify(value any, indent ...any) (string, error)
```
Encode the given *value* as a JSON string, optionally using *indent* to pretty format the output.

<a name="lower">
```go
lower(s string) string
```
Return a copy of string *s* with all Unicode letters mapped to their lower case.

<a name="ltrim">
```go
ltrim(s string, prefix string) string
```
Return a copy of string *s* with the leading *prefix* removed.

<a name="markdown">
```go
markdown(value any) (string, error)
```
Render the given Markdown string *value* as sanitized HTML.

<a name="percent">
```go
percent(value any, n ...any) (string, error)
```
Return the given floating point *value* as a percentage of *n*, or 100.0 if *n* is not specified.

<a name="replace">
```go
replace(s string, old string, new string, n int) string
```
Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.

<a name="rtrim">
```go
rtrim(s string, suffix string) string
```
Return a copy of string *s* with the trailing *suffix* removed.

<a name="rxreplace">
```go
rxreplace(s any, pattern string, repl string) (string, error)
```
Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.

<a name="split">
```go
split(s string, delimiter string, ...any)
```
Return a string array of elements resulting from *s* being split by *delimiter*, up to *n* times (if specified).

<a name="strcount">
```go
strcount(s string, substr string) int
```
Count *s* for the number of non-overlapping instances of *substr*.

<a name="surroundedBy">
```go
surroundedBy(s any, prefix string, suffix string) bool
```
Return whether string *s* starts with *prefix* and ends with *suffix*.

<a name="thousandify">
```go
thousandify(value any, sep ...any) string
```
Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.

<a name="titleize">
```go
titleize(s string) string
```
Return a copy of *s* with all Unicode letters that begin words mapped to their title case.

<a name="trim">
```go
trim(s string) string
```
Return a copy of *s* with all leading and trailing whitespace characters removed.

<a name="upper">
```go
upper(s string) string
```
Return a copy of *s* with all letters capitalized.

