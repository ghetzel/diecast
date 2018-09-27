# Diecast Function Reference

Diecast templates have access to a standard set of functions that aim to make working with data and
building web pages easier. Use the reference below to see which functions are available and how to
use them.

## Function List

- [add](#add)
- [addTime](#addTime)
- [ago](#ago)
- [any](#any)
- [apply](#apply)
- [asBool](#asBool)
- [asDuration](#asDuration)
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
- [colorFromValue](#colorFromValue)
- [colorToHSL](#colorToHSL)
- [colorToHex](#colorToHex)
- [colorToRGB](#colorToRGB)
- [compact](#compact)
- [contains](#contains)
- [count](#count)
- [csv](#csv)
- [darken](#darken)
- [dir](#dir)
- [dirname](#dirname)
- [divide](#divide)
- [duration](#duration)
- [elide](#elide)
- [eqx](#eqx)
- [extname](#extname)
- [extractTime](#extractTime)
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
- [head](#head)
- [headers](#headers)
- [hquery](#hquery)
- [hyphenate](#hyphenate)
- [increment](#increment)
- [incrementByValue](#incrementByValue)
- [indexOf](#indexOf)
- [intersect](#intersect)
- [ireverse](#ireverse)
- [isAfter](#isAfter)
- [isArray](#isArray)
- [isBefore](#isBefore)
- [isBool](#isBool)
- [isEmpty](#isEmpty)
- [isFloat](#isFloat)
- [isInt](#isInt)
- [isMap](#isMap)
- [isZero](#isZero)
- [isort](#isort)
- [join](#join)
- [jsonify](#jsonify)
- [keys](#keys)
- [last](#last)
- [leastcommon](#leastcommon)
- [lighten](#lighten)
- [lower](#lower)
- [ltrim](#ltrim)
- [mapify](#mapify)
- [markdown](#markdown)
- [mimeparams](#mimeparams)
- [mimetype](#mimetype)
- [mod](#mod)
- [mostcommon](#mostcommon)
- [multiply](#multiply)
- [murmur3](#murmur3)
- [negate](#negate)
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
- [rest](#rest)
- [reverse](#reverse)
- [reverse](#reverse)
- [round](#round)
- [rtrim](#rtrim)
- [rxreplace](#rxreplace)
- [sanitize](#sanitize)
- [sequence](#sequence)
- [set](#set)
- [shuffle](#shuffle)
- [since](#since)
- [sort](#sort)
- [split](#split)
- [strcount](#strcount)
- [stringify](#stringify)
- [stripHtml](#stripHtml)
- [subtract](#subtract)
- [sunrise](#sunrise)
- [sunset](#sunset)
- [surroundedBy](#surroundedBy)
- [tail](#tail)
- [templateKey](#templateKey)
- [thousandify](#thousandify)
- [time](#time)
- [titleize](#titleize)
- [trim](#trim)
- [tsv](#tsv)
- [underscore](#underscore)
- [uniq](#uniq)
- [uniqByKey](#uniqByKey)
- [uniqByKeyLast](#uniqByKeyLast)
- [unsafe](#unsafe)
- [upper](#upper)
- [uuid](#uuid)
- [uuidRaw](#uuidRaw)
- [values](#values)
- [var](#var)
## Function Usage

---

<a name="add"></a>
```
add values [any ..] -> float64
```
Return the sum of all of the given *values*.

---

<a name="addTime"></a>
```
addTime duration string at [any ..] -> Time error
```
Return a time with with given *duration* added to it.  Can specify time *at* to apply the change to.

---

<a name="ago"></a>
```
ago duration string [any ..] -> Time error
```
Return a Time subtracted by the given *duration*.

---

<a name="any"></a>
```
any input any wanted [any ..] -> bool
```
Return whether *input* array contains any of the the elements *wanted*.

---

<a name="apply"></a>
```
apply input any [any ..] ->  error
```
Apply a function to each of the elements in the given *input* array. Note that functions must accept one any-type argument.

---

<a name="asBool"></a>
```
asBool value any -> bool error
```
Attempt to convert the given *value* to a boolean value.

---

<a name="asDuration"></a>
```
asDuration value string -> Duration error
```
Attempt to parse the given *value* as a time duration.

---

<a name="asFloat"></a>
```
asFloat value any -> float64 error
```
Attempt to convert the given *value* to a floating-point number.

---

<a name="asInt"></a>
```
asInt value any -> int64 error
```
Attempt to convert the given *value* to an integer.

---

<a name="asStr"></a>
```
asStr value any -> string error
```
Return the *value* as a string.

---

<a name="asTime"></a>
```
asTime value any -> Time error
```
Attempt to parse the given *value* as a date/time value.

---

<a name="autobyte"></a>
```
autobyte bytesize any [any ..] -> string error
```
Attempt to convert the given *bytesize* number to a string representation of the value in bytes.

---

<a name="autotype"></a>
```
autotype value any -> 
```
Attempt to automatically determine the type if *value* and return the converted output.

---

<a name="base32"></a>
```
base32 input any -> string
```
Encode the *input* bytes with the Base32 encoding scheme.

---

<a name="base58"></a>
```
base58 input any -> string
```
Encode the *input* bytes with the Base58 (Bitcoin alphabet) encoding scheme.

---

<a name="base64"></a>
```
base64 input any [any ..] -> string
```
Encode the *input* bytes with the Base64 encoding scheme.  Optionally specify the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).

---

<a name="basename"></a>
```
basename path any -> string
```
Return the filename component of the given *path*.

---

<a name="camelize"></a>
```
camelize s any -> string
```
Return a copy of *s* transformed into CamelCase.

---

<a name="colorFromValue"></a>
```
colorFromValue any -> string
```
Consistently generate a color from a given value.

---

<a name="colorToHSL"></a>
```
colorToHSL any -> string error
```
Convert the given color to an "hsl()" or "hsla()" color specification.

---

<a name="colorToHex"></a>
```
colorToHex any -> string error
```
Convert the given color to a "#RRGGBB" or "#RRGGBBAA" color specification.

---

<a name="colorToRGB"></a>
```
colorToRGB any -> string error
```
Convert the given color to an "rgb()" or "rgba()" color specification.

---

<a name="compact"></a>
```
compact input any -> 
```
Return an copy of given *input* array with all zero-valued elements removed.

---

<a name="contains"></a>
```
contains s string substr string -> bool
```
Return whether a string *s* contains *substr*.

---

<a name="count"></a>
```
count len any -> int
```
A type-relaxed version of **len**.

---

<a name="csv"></a>
```
csv values any any -> string error
```
Render the given *values* as a line suitable for inclusion in a common-separated values file.

---

<a name="darken"></a>
```
darken any float64 -> string error
```
Darken the given color by a percent.

---

<a name="dir"></a>
```
dir path [any ..] ->  error
```
Return a list of files and directories in *path*, or in the current directory if not specified.

---

<a name="dirname"></a>
```
dirname path any -> string
```
Return the directory path component of the given *path*.

---

<a name="divide"></a>
```
divide values [any ..] -> float64 error
```
Sequentially divide all of the given *values*.

---

<a name="duration"></a>
```
duration value any unit string format [any ..] -> string error
```
Convert the given *value* from a duration of *unit* into the given time *format*.

---

<a name="elide"></a>
```
elide text any int -> string
```
Truncates the given *text* in a word-aware manner to the given number of characters.

---

<a name="eqx"></a>
```
eqx eq any any -> bool error
```
A relaxed-type version of the **eq** builtin function.

---

<a name="extname"></a>
```
extname path any -> string
```
Return the extension component of the given *path* (always prefixed with a dot [.]).

---

<a name="extractTime"></a>
```
extractTime any -> Time error
```
Attempt to extract a date from the given string

---

<a name="filter"></a>
```
filter input any expression string ->  error
```
Return the given *input* array with only elements where *expression* evaluates to a truthy value.

---

<a name="filterByKey"></a>
```
filterByKey input any string [any ..] ->  error
```
Return a subset of the elements in the *input* array whose map values contain the *key*, optionally matching *expression*.

---

<a name="findkey"></a>
```
findkey input any key string ->  error
```
Recursively scans the given *input* array or map and returns all values of the given *key*.

---

<a name="first"></a>
```
first input any ->  error
```
Return the first value from the given *input* array.

---

<a name="firstByKey"></a>
```
firstByKey input any string [any ..] ->  error
```
Return the first element in the *input* array whose map values contain the *key*, optionally matching *expression*.

---

<a name="flatten"></a>
```
flatten any -> 
```
Return an array of values with all nested subarrays merged into a single level.

---

<a name="get"></a>
```
get any string [any ..] -> 
```
Get a key from a map.

---

<a name="groupBy"></a>
```
groupBy input any string [any ..] ->  error
```
Return the given *input* array-of-objects as an object, keyed on the value of the specified group *field*.  The field argument can be a template.

---

<a name="has"></a>
```
has want any input any -> bool
```
Return whether *want* is an element of the given *input* array.

---

<a name="hasPrefix"></a>
```
hasPrefix s string prefix string -> bool
```
Return whether string *s* has the given *prefix*.

---

<a name="hasSuffix"></a>
```
hasSuffix s string suffix string -> bool
```
Return whether string *s* has the given *suffix*.

---

<a name="head"></a>
```
head input any n int -> 
```
Return from the *input* array the first *n* items.

---

<a name="headers"></a>
```
headers header string -> string
```
Return the value of the *header* HTTP request header from the request used to generate the current view.

---

<a name="hquery"></a>
```
hquery document any string ->  error
```
Queries a given HTML **document** (as returned by a Binding) and returns a list of Elements matching the given **selector**

---

<a name="hyphenate"></a>
```
hyphenate s any -> string
```
Return a copy of *s* transformed into hyphen-case.

---

<a name="increment"></a>
```
increment string [any ..] -> 
```
Increment a named variable by an amount.

---

<a name="incrementByValue"></a>
```
incrementByValue string any [any ..] -> 
```
Add a number to a counter tracking the number of occurrences of a specific value.

---

<a name="indexOf"></a>
```
indexOf input any value any -> int
```
Iterate through the *input* array and return the index of *value*, or -1 if not present.

---

<a name="intersect"></a>
```
intersect first any second any -> 
```
Return the intersection of the *first* and *second* slices.

---

<a name="ireverse"></a>
```
ireverse input any [any ..] -> 
```
Return the *input* array sorted in lexical descending order (case insensitive).

---

<a name="isAfter"></a>
```
isAfter first any second [any ..] -> bool error
```
Return whether the *first* time is after the *second* one.

---

<a name="isArray"></a>
```
isArray value any -> bool
```
Return whether the given *value* is an iterable array or slice.

---

<a name="isBefore"></a>
```
isBefore first any second [any ..] -> bool error
```
Return whether the *first* time is before the *second* one.

---

<a name="isBool"></a>
```
isBool value any -> bool
```
Return whether the given *value* is a boolean type.

---

<a name="isEmpty"></a>
```
isEmpty value any -> bool
```
Return whether the given *value* is empty.

---

<a name="isFloat"></a>
```
isFloat value any -> bool
```
Return whether the given *value* is a floating-point type.

---

<a name="isInt"></a>
```
isInt value any -> bool
```
Return whether the given *value* is an integer type.

---

<a name="isMap"></a>
```
isMap value any -> bool
```
Return whether the given *value* is a key-value map type.

---

<a name="isZero"></a>
```
isZero value any -> bool
```
Return whether the given *value* is an zero-valued variable.

---

<a name="isort"></a>
```
isort input any [any ..] -> 
```
Return the *input* array sorted in lexical ascending order (case insensitive).

---

<a name="join"></a>
```
join input any delimiter string -> string
```
Join the *input* array on *delimiter* and return a string.

---

<a name="jsonify"></a>
```
jsonify value any indent [any ..] -> string error
```
Encode the given *value* as a JSON string, optionally using *indent* to pretty format the output.

---

<a name="keys"></a>
```
keys input any -> 
```
Given an *input* map, return all of the keys.

---

<a name="last"></a>
```
last input any ->  error
```
Return the last value from the given *input* array.

---

<a name="leastcommon"></a>
```
leastcommon input any ->  error
```
Return element in the *input* array that appears the least frequently.

---

<a name="lighten"></a>
```
lighten any float64 -> string error
```
Lighten the given color by a percent.

---

<a name="lower"></a>
```
lower s string -> string
```
Return a copy of string *s* with all Unicode letters mapped to their lower case.

---

<a name="ltrim"></a>
```
ltrim s any prefix string -> string
```
Return a copy of string *s* with the leading *prefix* removed.

---

<a name="mapify"></a>
```
mapify any -> 
```
Convert the input value into a map.

---

<a name="markdown"></a>
```
markdown value any -> HTML error
```
Render the given Markdown string *value* as sanitized HTML.

---

<a name="mimeparams"></a>
```
mimeparams string -> 
```
Returns the parameters portion of the MIME type of the given filename

---

<a name="mimetype"></a>
```
mimetype string -> string
```
Returns a best guess MIME type for the given filename

---

<a name="mod"></a>
```
mod values [any ..] -> float64 error
```
Return the modulus of all of the given *values*.

---

<a name="mostcommon"></a>
```
mostcommon input any ->  error
```
Return element in the *input* array that appears the most frequently.

---

<a name="multiply"></a>
```
multiply values [any ..] -> float64
```
Return the product of all of the given *values*.

---

<a name="murmur3"></a>
```
murmur3 input any -> uint64 error
```
hash the *input* data using the Murmur 3 algorithm.

---

<a name="negate"></a>
```
negate any -> float64
```
Return the given number multiplied by -1

---

<a name="nex"></a>
```
nex ne any any -> bool error
```
A relaxed-type version of the **ne** builtin function.

---

<a name="now"></a>
```
now format [any ..] -> string error
```
Return the current time formatted using *format*.  See [Time Formats](#time-formats) for acceptable formats.

---

<a name="param"></a>
```
param any -> 
```
Return the value of the named or indexed URL parameter, or nil of none are present.

---

<a name="pathjoin"></a>
```
pathjoin values [any ..] -> string
```
Return the value of all *values* join on the system path separator.

---

<a name="payload"></a>
```
payload [any ..] -> 
```
Return the body supplied with the request used to generate the current view.

---

<a name="percent"></a>
```
percent value any n [any ..] -> string error
```
Return the given floating point *value* as a percentage of *n*, or 100.0 if *n* is not specified.

---

<a name="pluck"></a>
```
pluck input any key string [any ..] -> 
```
Given an *input* array of maps, retrieve the values of *key* from all elements.

---

<a name="pop"></a>
```
pop name string -> 
```
Remove the last item from *name* and return it.

---

<a name="pow"></a>
```
pow values [any ..] -> float64 error
```
Sequentially exponentiate of all of the given *values*.

---

<a name="push"></a>
```
push name string value [any ..] -> 
```
Append to variable *name* to *value*.

---

<a name="pwd"></a>
```
pwd  -> string error
```
Return the present working directory

---

<a name="qs"></a>
```
qs key any fallback [any ..] -> 
```
Return the value of query string parameter *key* in the current URL, or return *fallback*.

---

<a name="querystrings"></a>
```
querystrings  -> 
```
Return a map of all of the query string parameters in the current URL.

---

<a name="random"></a>
```
random n int ->  error
```
Return a random array of *n* bytes. The random source used is suitable for cryptographic purposes.

---

<a name="replace"></a>
```
replace s string old string new string n int -> string
```
Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.

---

<a name="rest"></a>
```
rest input any ->  error
```
Return all but the first value from the given *input* array.

---

<a name="reverse"></a>
```
reverse input any [any ..] -> 
```
Return the *input* array sorted in lexical descending order.

---

<a name="reverse"></a>
```
reverse array any [any ..] -> 
```
Return the given *array* in reverse order.

---

<a name="round"></a>
```
round any [any ..] -> float64 error
```
Round a number to the nearest n places.

---

<a name="rtrim"></a>
```
rtrim s any suffix string -> string
```
Return a copy of string *s* with the trailing *suffix* removed.

---

<a name="rxreplace"></a>
```
rxreplace s any pattern string repl string -> string error
```
Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.

---

<a name="sanitize"></a>
```
sanitize string -> HTML
```
Takes a raw HTML string and santizes it, removing attributes and elements that can be used to evaluate scripts, but leaving the rest.  Useful for preparing user-generated HTML for display.

---

<a name="sequence"></a>
```
sequence n any -> 
```
Return an array of integers representing a sequence from [0, *n*).

---

<a name="set"></a>
```
set name string key string value [any ..] -> 
```
Treat the runtime variable *name* as a map, setting *key* to *value*.

---

<a name="shuffle"></a>
```
shuffle input [any ..] -> 
```
Return the *input* array with the elements rearranged in random order.

---

<a name="since"></a>
```
since time any [any ..] -> Duration error
```
Return the amount of time that has elapsed since *time*, optionally rounded to the nearest *interval*.

---

<a name="sort"></a>
```
sort input any [any ..] -> 
```
Return the *input* array sorted in lexical ascending order.

---

<a name="split"></a>
```
split s string delimiter string [any ..] -> 
```
Return a string array of elements resulting from *s* being split by *delimiter*, up to *n* times (if specified).

---

<a name="strcount"></a>
```
strcount s string substr string -> int
```
Count *s* for the number of non-overlapping instances of *substr*.

---

<a name="stringify"></a>
```
stringify input any -> 
```
Return the given *input* array with all values converted to strings.

---

<a name="stripHtml"></a>
```
stripHtml input any -> string
```
strips HTML tags from the given *input* text, leaving the text content behind.

---

<a name="subtract"></a>
```
subtract values [any ..] -> float64
```
Sequentially subtract all of the given *values*.

---

<a name="sunrise"></a>
```
sunrise float64 float64 [any ..] -> Time error
```
Return the time of apparent sunrise at the given coordinates, optionally for a given time.

---

<a name="sunset"></a>
```
sunset float64 float64 [any ..] -> Time error
```
Return the time of apparent sunset at the given coordinates, optionally for a given time.

---

<a name="surroundedBy"></a>
```
surroundedBy s any prefix string suffix string -> bool
```
Return whether string *s* starts with *prefix* and ends with *suffix*.

---

<a name="tail"></a>
```
tail input any n int -> 
```
Return from the *input* array the last *n* items.

---

<a name="templateKey"></a>
```
templateKey file any key any [any ..] ->  error
```
Open the given *file* and retrieve the *key* from the page object in its header.

---

<a name="thousandify"></a>
```
thousandify value any sep [any ..] -> string
```
Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.

---

<a name="time"></a>
```
time format any [any ..] -> string error
```
Return the given Time formatted using *format*.  See [Time Formats](#time-formats) for acceptable formats.

---

<a name="titleize"></a>
```
titleize s string -> string
```
Return a copy of *s* with all Unicode letters that begin words mapped to their title case.

---

<a name="trim"></a>
```
trim s string -> string
```
Return a copy of *s* with all leading and trailing whitespace characters removed.

---

<a name="tsv"></a>
```
tsv values any any -> string error
```
Render the given *values* as a line suitable for inclusion in a tab-separated values file.

---

<a name="underscore"></a>
```
underscore s any -> string
```
Return a copy of *s* transformed into snake_case.

---

<a name="uniq"></a>
```
uniq input any -> 
```
Return an array of unique values from the given *input* array.

---

<a name="uniqByKey"></a>
```
uniqByKey input any string [any ..] ->  error
```
Return a subset of the elements in the *input* array whose map values are unique for all values of *key*, preserving the first duplicate value. Values are optionally preprocessed using *expression*.

---

<a name="uniqByKeyLast"></a>
```
uniqByKeyLast input any string [any ..] ->  error
```
Return a subset of the elements in the *input* array whose map values are unique for all values of *key*, preserving the last duplicate value. Values are optionally preprocessed using *expression*.

---

<a name="unsafe"></a>
```
unsafe string -> HTML
```
Return an unescaped raw HTML segment for direct inclusion in the rendered template output.  This is a common antipattern that leads to all kinds of security issues from poorly-constrained implementations, so you are forced to acknowledge this by typing "unsafe".

---

<a name="upper"></a>
```
upper s string -> string
```
Return a copy of *s* with all letters capitalized.

---

<a name="uuid"></a>
```
uuid  -> string
```
Generate a new Version 4 UUID.

---

<a name="uuidRaw"></a>
```
uuidRaw  -> 
```
Generate the raw bytes of a new Version 4 UUID.

---

<a name="values"></a>
```
values input any -> 
```
Given an *input* map, return all of the values.

---

<a name="var"></a>
```
var name string value [any ..] -> 
```
Set the runtime variable *name* to *value*.

# Time Formats

A special string can be used to specify how a given date value should be formatted for display.
Diecast supports the same syntax as Golang's [time.Format](https://golang.org/pkg/time/#pkg-constants)
function, as well as additional commonly-used formats.

## Predefined Formats

| Format String | Example                               |
| ------------- | ------------------------------------- |
| `kitchen`     | "3:04PM"                              |
| `timer`       | "15:04:05"                            |
| `rfc3339`     | "2006-01-02T15:04:05Z07:00"           |
| `rfc3339ns`   | "2006-01-02T15:04:05.999999999Z07:00" |
| `rfc822`      | "02 Jan 06 15:04 MST"                 |
| `rfc822z`     | "02 Jan 06 15:04 -0700"               |
| `epoch`       | "1136239445"                          |
| `epoch-ms`    | "1136239445999"                       |
| `epoch-us`    | "1136239445999999"                    |
| `epoch-ns`    | "1136239445999999999"                 |
| `day`         | "Monday"                              |
| `slash`       | "01/02/2006"                          |
| `slash-dmy`   | "02/01/2006"                          |
| `ymd`         | "2006-01-02"                          |
| `ruby`        | "Mon Jan 02 15:04:05 -0700 2006"      |

## Custom Formats

You can also specify a custom format string by using the components the the _reference date_ as an
example to Diecast on how to translate the given date into the output you want.  The reference
date/time is: `Mon Jan 2 15:04:05 MST 2006`.  In the predefined formats above, the examples given all
use this reference date/time, and you can refer to those formats for building your own strings.

For example, given the date 2018-03-10T16:30:00, and the custom format string "Mon, January _1, 2006",
that date would be displayed as "Sat, March 10, 2018".  The format was built by providing examples
from the reference date on how to do the conversion.  The values used in the reference date have been
carefully chosen to avoid any ambiguity when specifying custom formats.

