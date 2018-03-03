# Diecast Function Reference

Diecast templates have access to a standard set of functions that aim to make working with data and
building web pages easier. Use the reference below to see which functions are available and how to
use them.

## Function List

- [add](#add)
- [ago](#ago)
- [any](#any)
- [asBool](#asBool)
- [asFloat](#asFloat)
- [asInt](#asInt)
- [asStr](#asStr)
- [asTime](#asTime)
- [autobyte](#autobyte)
- [autotype](#autotype)
- [base32](#base32)
- [base58](#base58)
- [base64](#base64)
- [basename](#basename)
- [camelize](#camelize)
- [compact](#compact)
- [contains](#contains)
- [count](#count)
- [csv](#csv)
- [dir](#dir)
- [dirname](#dirname)
- [divide](#divide)
- [duration](#duration)
- [elide](#elide)
- [eqx](#eqx)
- [extname](#extname)
- [filter](#filter)
- [filterByKey](#filterByKey)
- [findkey](#findkey)
- [first](#first)
- [firstByKey](#firstByKey)
- [flatten](#flatten)
- [get](#get)
- [groupBy](#groupBy)
- [has](#has)
- [hasPrefix](#hasPrefix)
- [hasSuffix](#hasSuffix)
- [headers](#headers)
- [hquery](#hquery)
- [indexOf](#indexOf)
- [ireverse](#ireverse)
- [isArray](#isArray)
- [isBool](#isBool)
- [isEmpty](#isEmpty)
- [isFloat](#isFloat)
- [isInt](#isInt)
- [isMap](#isMap)
- [isZero](#isZero)
- [isort](#isort)
- [join](#join)
- [jsonify](#jsonify)
- [last](#last)
- [leastcommon](#leastcommon)
- [lower](#lower)
- [ltrim](#ltrim)
- [markdown](#markdown)
- [mimeparams](#mimeparams)
- [mimetype](#mimetype)
- [mod](#mod)
- [mostcommon](#mostcommon)
- [multiply](#multiply)
- [murmur3](#murmur3)
- [nex](#nex)
- [now](#now)
- [param](#param)
- [pathjoin](#pathjoin)
- [payload](#payload)
- [percent](#percent)
- [pluck](#pluck)
- [pop](#pop)
- [pow](#pow)
- [push](#push)
- [pwd](#pwd)
- [qs](#qs)
- [querystrings](#querystrings)
- [random](#random)
- [replace](#replace)
- [reverse](#reverse)
- [rtrim](#rtrim)
- [rxreplace](#rxreplace)
- [sanitize](#sanitize)
- [sequence](#sequence)
- [since](#since)
- [sort](#sort)
- [split](#split)
- [strcount](#strcount)
- [stringify](#stringify)
- [stripHtml](#stripHtml)
- [subtract](#subtract)
- [surroundedBy](#surroundedBy)
- [thousandify](#thousandify)
- [time](#time)
- [titleize](#titleize)
- [trim](#trim)
- [tsv](#tsv)
- [underscore](#underscore)
- [uniq](#uniq)
- [unsafe](#unsafe)
- [upper](#upper)
- [uuid](#uuid)
- [uuidRaw](#uuidRaw)
- [var](#var)
## Function Usage

---

<a name="add"></a>
```go
add(values ...any) float64
```
Return the sum of all of the given *values*.

---

<a name="ago"></a>
```go
ago(duration string, ...any) (Time, error)
```
Return a Time subtracted by the given *duration*.

---

<a name="any"></a>
```go
any(input any, wanted ...any) bool
```
Return whether *input* array contains any of the the elements *wanted*.

---

<a name="asBool"></a>
```go
asBool(value any) (bool, error)
```
Attempt to convert the given *value* to a boolean value.

---

<a name="asFloat"></a>
```go
asFloat(value any) (float64, error)
```
Attempt to convert the given *value* to a floating-point number.

---

<a name="asInt"></a>
```go
asInt(value any) (int64, error)
```
Attempt to convert the given *value* to an integer.

---

<a name="asStr"></a>
```go
asStr(value any) (string, error)
```
Return the *value* as a string.

---

<a name="asTime"></a>
```go
asTime(value any) (Time, error)
```
Attempt to parse the given *value* as a date/time value.

---

<a name="autobyte"></a>
```go
autobyte(bytesize any, ...any) (string, error)
```
Attempt to convert the given *bytesize* number to a string representation of the value in bytes.

---

<a name="autotype"></a>
```go
autotype(value any)
```
Attempt to automatically determine the type if *value* and return the converted output.

---

<a name="base32"></a>
```go
base32(input any) string
```
Encode the *input* bytes with the Base32 encoding scheme.

---

<a name="base58"></a>
```go
base58(input any) string
```
Encode the *input* bytes with the Base58 (Bitcoin alphabet) encoding scheme.

---

<a name="base64"></a>
```go
base64(input any, ...any) string
```
Encode the *input* bytes with the Base64 encoding scheme.  Optionally specify the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).

---

<a name="basename"></a>
```go
basename(path any) string
```
Return the filename component of the given *path*.

---

<a name="camelize"></a>
```go
camelize(s any) string
```
Return a copy of *s* transformed into CamelCase.

---

<a name="compact"></a>
```go
compact(input any)
```
Return an copy of given *input* array with all zero-valued elements removed.

---

<a name="contains"></a>
```go
contains(s string, substr string) bool
```
Return whether a string *s* contains *substr*.

---

<a name="count"></a>
```go
count(len any) int
```
A type-relaxed version of **len**.

---

<a name="csv"></a>
```go
csv(values any, any) (string, error)
```
Render the given *values* as a line suitable for inclusion in a common-separated values file.

---

<a name="dir"></a>
```go
dir(path ...any) (, error)
```
Return a list of files and directories in *path*, or in the current directory if not specified.

---

<a name="dirname"></a>
```go
dirname(path any) string
```
Return the directory path component of the given *path*.

---

<a name="divide"></a>
```go
divide(values ...any) (float64, error)
```
Sequentially divide all of the given *values*.

---

<a name="duration"></a>
```go
duration(value any, unit string, format ...any) (string, error)
```
Convert the given *value* from a duration of *unit* into the given time *format*.

---

<a name="elide"></a>
```go
elide(text any, int) string
```
Truncates the given *text* in a word-aware manner to the given number of characters.

---

<a name="eqx"></a>
```go
eqx(eq any, any) (bool, error)
```
A relaxed-type version of the **eq** builtin function.

---

<a name="extname"></a>
```go
extname(path any) string
```
Return the extension component of the given *path* (always prefixed with a dot [.]).

---

<a name="filter"></a>
```go
filter(input any, expression string) (, error)
```
Return the given *input* array with only elements where *expression* evaluates to a truthy value.

---

<a name="filterByKey"></a>
```go
filterByKey(input any, string, ...any) (, error)
```
Return a subset of the elements in the *input* array whose map values contain the *key*, optionally matching *expression*.

---

<a name="findkey"></a>
```go
findkey(input any, key string) (, error)
```
Recursively scans the given *input* array or map and returns all values of the given *key*.

---

<a name="first"></a>
```go
first(input any) (, error)
```
Return the first value from the given *input* array.

---

<a name="firstByKey"></a>
```go
firstByKey(input any, string, ...any) (, error)
```
Return the first elements in the *input* array whose map values contain the *key*, optionally matching *expression*.

---

<a name="flatten"></a>
```go
flatten(any)
```
Return an array of values with all nested subarrays merged into a single level.

---

<a name="get"></a>
```go
get(any, string, ...any)
```
Get a key from a map.

---

<a name="groupBy"></a>
```go
groupBy(input any, string, ...any) (, error)
```
Return the given *input* array-of-objects as an object, keyed on the value of the specified group *field*.  The field argument can be a template.

---

<a name="has"></a>
```go
has(want any, input any) bool
```
Return whether *want* is an element of the given *input* array.

---

<a name="hasPrefix"></a>
```go
hasPrefix(s string, prefix string) bool
```
Return whether string *s* has the given *prefix*.

---

<a name="hasSuffix"></a>
```go
hasSuffix(s string, suffix string) bool
```
Return whether string *s* has the given *suffix*.

---

<a name="headers"></a>
```go
headers(header string) string
```
Return the value of the *header* HTTP request header from the request used to generate the current view.

---

<a name="hquery"></a>
```go
hquery(document any, string) (, error)
```
Queries a given HTML **document** (as returned by a Binding) and returns a list of Elements matching the given **selector**

---

<a name="indexOf"></a>
```go
indexOf(input any, value any) int
```
Iterate through the *input* array and return the index of *value*, or -1 if not present.

---

<a name="ireverse"></a>
```go
ireverse(input any, ...any)
```
Return the *input* array sorted in lexical descending order (case insensitive).

---

<a name="isArray"></a>
```go
isArray(value any) bool
```
Return whether the given *value* is an iterable array or slice.

---

<a name="isBool"></a>
```go
isBool(value any) bool
```
Return whether the given *value* is a boolean type.

---

<a name="isEmpty"></a>
```go
isEmpty(value any) bool
```
Return whether the given *value* is empty.

---

<a name="isFloat"></a>
```go
isFloat(value any) bool
```
Return whether the given *value* is a floating-point type.

---

<a name="isInt"></a>
```go
isInt(value any) bool
```
Return whether the given *value* is an integer type.

---

<a name="isMap"></a>
```go
isMap(value any) bool
```
Return whether the given *value* is a key-value map type.

---

<a name="isZero"></a>
```go
isZero(value any) bool
```
Return whether the given *value* is an zero-valued variable.

---

<a name="isort"></a>
```go
isort(input any, ...any)
```
Return the *input* array sorted in lexical ascending order (case insensitive).

---

<a name="join"></a>
```go
join(input any, delimiter string) string
```
Join the *input* array on *delimiter* and return a string.

---

<a name="jsonify"></a>
```go
jsonify(value any, indent ...any) (string, error)
```
Encode the given *value* as a JSON string, optionally using *indent* to pretty format the output.

---

<a name="last"></a>
```go
last(input any) (, error)
```
Return the last value from the given *input* array.

---

<a name="leastcommon"></a>
```go
leastcommon(input any) (, error)
```
Return element in the *input* array that appears the least frequently.

---

<a name="lower"></a>
```go
lower(s string) string
```
Return a copy of string *s* with all Unicode letters mapped to their lower case.

---

<a name="ltrim"></a>
```go
ltrim(s string, prefix string) string
```
Return a copy of string *s* with the leading *prefix* removed.

---

<a name="markdown"></a>
```go
markdown(value any) (HTML, error)
```
Render the given Markdown string *value* as sanitized HTML.

---

<a name="mimeparams"></a>
```go
mimeparams(string)
```
Returns the parameters portion of the MIME type of the given filename

---

<a name="mimetype"></a>
```go
mimetype(string) string
```
Returns a best guess MIME type for the given filename

---

<a name="mod"></a>
```go
mod(values ...any) (float64, error)
```
Return the modulus of all of the given *values*.

---

<a name="mostcommon"></a>
```go
mostcommon(input any) (, error)
```
Return element in the *input* array that appears the most frequently.

---

<a name="multiply"></a>
```go
multiply(values ...any) float64
```
Return the product of all of the given *values*.

---

<a name="murmur3"></a>
```go
murmur3(input any) (uint64, error)
```
hash the *input* data using the Murmur 3 algorithm.

---

<a name="nex"></a>
```go
nex(ne any, any) (bool, error)
```
A relaxed-type version of the **ne** builtin function.

---

<a name="now"></a>
```go
now(format ...any) (string, error)
```
Return the current time formatted using *format*.  See [Time Formats](#time-formats) for acceptable formats.

---

<a name="param"></a>
```go
param(any)
```
Return the value of the named or indexed URL parameter, or nil of none are present.

---

<a name="pathjoin"></a>
```go
pathjoin(values ...any) string
```
Return the value of all *values* join on the system path separator.

---

<a name="payload"></a>
```go
payload(...any)
```
Return the body supplied with the request used to generate the current view.

---

<a name="percent"></a>
```go
percent(value any, n ...any) (string, error)
```
Return the given floating point *value* as a percentage of *n*, or 100.0 if *n* is not specified.

---

<a name="pluck"></a>
```go
pluck(input any, key string)
```
Given an *input* array of maps, retrieve the values of *key* from all elements.

---

<a name="pop"></a>
```go
pop(name string)
```
Remove the last item from *name* and return it.

---

<a name="pow"></a>
```go
pow(values ...any) (float64, error)
```
Sequentially exponentiate of all of the given *values*.

---

<a name="push"></a>
```go
push(name string, value ...any)
```
Append to variable *name* to *value*.

---

<a name="pwd"></a>
```go
pwd() (string, error)
```
Return the present working directory

---

<a name="qs"></a>
```go
qs(key any, fallback ...any)
```
Return the value of query string parameter *key* in the current URL, or return *fallback*.

---

<a name="querystrings"></a>
```go
querystrings()
```
Return a map of all of the query string parameters in the current URL.

---

<a name="random"></a>
```go
random(n int) (, error)
```
Return a random array of *n* bytes. The random source used is suitable for cryptographic purposes.

---

<a name="replace"></a>
```go
replace(s string, old string, new string, n int) string
```
Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.

---

<a name="reverse"></a>
```go
reverse(input any, ...any)
```
Return the *input* array sorted in lexical descending order.

---

<a name="rtrim"></a>
```go
rtrim(s string, suffix string) string
```
Return a copy of string *s* with the trailing *suffix* removed.

---

<a name="rxreplace"></a>
```go
rxreplace(s any, pattern string, repl string) (string, error)
```
Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.

---

<a name="sanitize"></a>
```go
sanitize(string) HTML
```
Takes a raw HTML string and santizes it, removing attributes and elements that can be used to evaluate scripts, but leaving the rest.  Useful for preparing user-generated HTML for display.

---

<a name="sequence"></a>
```go
sequence(n any)
```
Return an array of integers representing a sequence from [0, *n*).

---

<a name="since"></a>
```go
since(time any, ...any) (Duration, error)
```
Return the amount of time that has elapsed since *time*, optionally rounded to the nearest *interval*.

---

<a name="sort"></a>
```go
sort(input any, ...any)
```
Return the *input* array sorted in lexical ascending order.

---

<a name="split"></a>
```go
split(s string, delimiter string, ...any)
```
Return a string array of elements resulting from *s* being split by *delimiter*, up to *n* times (if specified).

---

<a name="strcount"></a>
```go
strcount(s string, substr string) int
```
Count *s* for the number of non-overlapping instances of *substr*.

---

<a name="stringify"></a>
```go
stringify(input any)
```
Return the given *input* array with all values converted to strings.

---

<a name="stripHtml"></a>
```go
stripHtml(input any) string
```
strips HTML tags from the given *input* text, leaving the text content behind.

---

<a name="subtract"></a>
```go
subtract(values ...any) float64
```
Sequentially subtract all of the given *values*.

---

<a name="surroundedBy"></a>
```go
surroundedBy(s any, prefix string, suffix string) bool
```
Return whether string *s* starts with *prefix* and ends with *suffix*.

---

<a name="thousandify"></a>
```go
thousandify(value any, sep ...any) string
```
Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.

---

<a name="time"></a>
```go
time(format any, ...any) (string, error)
```
Return the given Time formatted using *format*.  See [Time Formats](#time-formats) for acceptable formats.

---

<a name="titleize"></a>
```go
titleize(s string) string
```
Return a copy of *s* with all Unicode letters that begin words mapped to their title case.

---

<a name="trim"></a>
```go
trim(s string) string
```
Return a copy of *s* with all leading and trailing whitespace characters removed.

---

<a name="tsv"></a>
```go
tsv(values any, any) (string, error)
```
Render the given *values* as a line suitable for inclusion in a tab-separated values file.

---

<a name="underscore"></a>
```go
underscore(s any) string
```
Return a copy of *s* transformed into snake_case.

---

<a name="uniq"></a>
```go
uniq(input any)
```
Return an array of unique values from the given *input* array.

---

<a name="unsafe"></a>
```go
unsafe(string) HTML
```
Return an unescaped raw HTML segment for direct inclusion in the rendered template output.  This is a common antipattern that leads to all kinds of security issues from poorly-constrained implementations, so you are forced to acknowledge this by typing "unsafe".

---

<a name="upper"></a>
```go
upper(s string) string
```
Return a copy of *s* with all letters capitalized.

---

<a name="uuid"></a>
```go
uuid() string
```
Generate a new Version 4 UUID.

---

<a name="uuidRaw"></a>
```go
uuidRaw()
```
Generate the raw bytes of a new Version 4 UUID.

---

<a name="var"></a>
```go
var(name string, value ...any)
```
Set the runtime variable *name* to *value*.

# Time Formats

