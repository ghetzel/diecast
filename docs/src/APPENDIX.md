### Time Formats

A special string can be used to specify how a given date value should be formatted for display.
Diecast supports the same syntax as Golang's [time.Format](https://golang.org/pkg/time/#pkg-constants)
function, as well as additional commonly-used formats.

#### Predefined Formats

| Format String | Example                               |
| ------------- | ------------------------------------- |
| `kitchen`     | "3:04PM"                              |
| `timer`       | "15:04:05"                            |
| `rfc3339`     | "2006-01-02T15:04:05Z07:00"           |
| `rfc3339ns`   | "2006-01-02T15:04:05.999999999Z07:00" |
| `rfc822`      | "02 Jan 06 15:04 MST"                 |
| `rfc822z`     | "02 Jan 06 15:04 -0700"               |
| `rfc1123`     | "Mon, 02 Jan 2006 15:04:05 MST"       |
| `rfc1123z`    | "Mon, 02 Jan 2006 15:04:05 -0700"     |
| `epoch`       | "1136239445"                          |
| `epoch-ms`    | "1136239445999"                       |
| `epoch-us`    | "1136239445999999"                    |
| `epoch-ns`    | "1136239445999999999"                 |
| `day`         | "Monday"                              |
| `slash`       | "01/02/2006"                          |
| `slash-dmy`   | "02/01/2006"                          |
| `ymd`         | "2006-01-02"                          |
| `ruby`        | "Mon Jan 02 15:04:05 -0700 2006"      |
| `ansic`       | "Mon Jan \_2 15:04:05 2006"           |
| `ruby`        | "Mon Jan 02 15:04:05 -0700 2006"      |
| `unixdate`    | "Mon Jan \_2 15:04:05 MST 2006"       |
| `stamp`       | "Jan \_2 15:04:05"                    |
| `stamp-ms`    | "Jan \_2 15:04:05.000"                |
| `stamp-us`    | "Jan \_2 15:04:05.000000"             |
| `stamp-ns`    | "Jan \_2 15:04:05.000000000"          |

#### Custom Formats

You can also specify a custom format string by using the components the the _reference date_ as an
example to Diecast on how to translate the given date into the output you want. The reference
date/time is: `Mon Jan 2 15:04:05 MST 2006`. In the predefined formats above, the examples given all
use this reference date/time, and you can refer to those formats for building your own strings.

For example, given the date 2018-03-10T16:30:00, and the custom format string "Mon, January \_1, 2006",
that date would be displayed as "Sat, March 10, 2018". The format was built by providing examples
from the reference date on how to do the conversion. The values used in the reference date have been
carefully chosen to avoid any ambiguity when specifying custom formats.

### Wildcard Patterns

Throughout the application, certain configuration options support a way to specify wildcard patterns that match more than just a specific string. These types of patterns are especially useful when working with URLs, where it is common to want settings to apply to all URLs that start with or otherwise contain a specific string. Below is a description of the wildcard support Diecast implements:

| Pattern | Example   | Description                                          |
| ------- | --------- | ---------------------------------------------------- |
| `*`     | `/api/*`  | Match zero or more _non-separator_ characters (`/`). |
| `**`    | `/api/**` | Match zero or more characters, including separators. |
| `?`     | `/api/?`  | Match zero or one character.                         |
