# Diecast [![GoDoc](https://godoc.org/github.com/ghetzel/diecast?status.svg)](https://godoc.org/github.com/ghetzel/diecast)
## Introduction

Diecast is a web server that allows you to render a directory tree of template files into HTML, CSS or anything other text-based media in real-time.  Data can be retrieved from remote sources during the template rendering process, creating dynamic web pages built by consuming APIs and remote files without the need to use client-side Javascript/AJAX calls or an intermediate server framework.

## Installation
### Golang / via `go get`

```
go get github.com/ghetzel/diecast/diecast
```

### macOS / Homebrew
### Windows
### Linux
### FreeBSD
### Binaries
### from Source

## Getting Started

Building a site using Diecast begins (and, to some extent, ends) with putting files in a directory.  When the `diecast` command is run in this directory, a local production-ready webserver will be started and the contents of the directory will be served.  If no other filenames or paths are requested, Diecast will look for and attempt to serve the file `index.html`.

## URL Structure

Diecast does not have a concept of URL path routing, but rather strives to enforce simple, linear hierachies by exposing the working directory directly as routable paths.  For example, if a user visits the path `/users/list`, Diecast will look for files to serve in the following order:

* `./users/list/index.html`
* `./users/list.html`
* `./_errors/404.html`
* `./_errors/4xx.html`
* `./_errors/default.html`

The first matching file from the list above will be served.

## Configuration

Diecast is configured by placing a file called `diecast.yml` in the same directory that the `diecast` command is run, or by specifying a filename with the `--config` command line flag.  This file is used to adjust how Diecast renders templates and when, as well as setting up options for how files are accessed and from where.  For more details, see the [Example Config File](/examples/diecast.sample.yml).

## Templating

Beyond merely acting as a simple file server, Diecast comes with a rich templating environment that allows for complex sites to be built in a composable way.  The default templating language used by Diecast is [Golang's built-in `text/template` package.](https://golang.org/pkg/text/template/).  Templates are plain text files that reside in the working directory, and consist of the template content, and optionally a header section called _front matter_.  These headers are used to specify template-specific data such as predefined data structures, paths of other templates to include, rendering options, and the inclusion of remote data via [bindings](#Bindings).  An example template looks like this:

```
---
layout: mobile-v1

bindings:
-   name:     members
    resource: /api/members.json

postprocessors:
- trim-empty-lines
- prettify-html
---
<!DOCTYPE html>
<html>
<body>
    <ul>
    {{ range $member := $.bindings.members }}
        <li>{{ $member }}</li>
    {{ end }}
    </ul>
</body>
</html>
```

### Language Overview

Golang's `text/template` package provides a syntactically-familiar and highly performant templating language.  When rendering HTML, CSS, or Javascript documents, the `html/template` parser is used.  This is the exact same language, but offers extensive context-aware automatic code escaping capabilities that ensure the output is safe against many common code injection techniques.  This is especially useful when using templates to render user-defined input.

#### Intro to `text/template`

The built-in templating language should be familiar to those coming from a background in other templating languages like [Jekyll](https://jekyllrb.com/), [Jinja2](http://jinja.pocoo.org/docs/2.10/), and [Mustache](https://mustache.github.io/).  Below is a quick guide on the high-level language constructs.  For detailed information, check out the [Golang `text/template` Language Overview](https://golang.org/pkg/text/template/#pkg-overview).

##### Output Text

```
Hello {{ $name }}! Today is {{ $date }}.
```

##### Conditionals (if/else if/else)

```
{{ if $pending }}
Access Pending
{{ else if $allowed }}
Access Granted
{{ else }}
Access Denied
{{ end }}
```

##### Loops

```
<h2>Members:</h2>
<ul>
{{ range $name := $names }}
    <li>{{ $name }}</li>
{{ end }}
</ul>

<h2>Ranks:</h2>
<ul>
{{ range $i, $name := $rankedNames }}
    <li>{{ $i }}: {{ $name }}</li>
{{ end }}
</ul>
```

##### Functions

```
Today is {{ now "ymd" }}, at {{ now "timer" }}.

There are {{ count $names }} members.
```

### Layouts

In addition to rendering individual files as standalone pages, Diecast also supports layouts.  Layouts serve as wrapper templates for the files being rendered in a directory tree.  Their primary purpose is to eliminate copying and boilerplate code.  Layouts are stored in a top-level directory called `_layouts`.  If the layout `_layouts/default.html` is present, it will automatically be used by default (e.g.: without explicitly specifying it in the Front Matter) on all pages.  The layout for any page can be specified in the `layout` Front Matter property, and the special value `layout: none` will disable layouts entirely for that page.

### Page Object

Diecast defines a global data structure in the `$.page` variable that can be used to provide site-wide values to templates.  The `page` structure can be defined in multiple places, allowing for the flexible expression of hierarchical data when rendering templates.  The `page` structure is inherited by child templates when rendering, and all values are deeply-merged together to form a single data structure for the template(s) to use.  For example, given the following files:

```yaml
# diecast.yml
header:
    page:
        site_title: WELCOME TO MY WEBSITE
```

```
---
# _layouts/default.html
page:
    colors:
    - red
    - green
    - blue
---
<html>
<head>
    <title>{{ if $.page.title }}{{ $.page.title }} :: {{ end }}{{ $.page.site_title }}</title>
</head>
<body>
    {{ template "content" . }}
</body>
</html>
```

```
---
# index.html
page:
    title: Home
---
<h1>Hello World!</h1>
<ul>
    {{ range $color := $.page.colors }}
    <li style="color: {{ $color }}">{{ $color }}</li>
    {{ end }}
</ul>
```

The final `page` data structure would look like this immediately before rendering `index.html`:

```yaml
page:
    site_title: WELCOME TO MY WEBSITE
    colors:
    - red
    - green
    - blue
    title: Home
```

...and the rendered output for `index.html` would look like this:

```html
<html>
<head>
    <title>Home :: WELCOME TO MY WEBSITE</title>
</head>
<body>
    <h1>Hello World!</h1>
    <ul>
        <li style="color: red">red</li>
        <li style="color: green">green</li>
        <li style="color: blue">blue</li>
    </ul>
</body>
</html>
```

### Bindings

Bindings are one of the most important concepts in Diecast.  Bindings (short for _data bindings_) are directives added to the Front Matter of layouts and templates that specify remote URLs to retrieve (via an HTTP client built in to `diecast`), as well as how to handle parsing the response data and what to do about errors.  This concept is extremely powerful, in that it allows you to create complex data-driven sites easily and cleanly by treating remote data from RESTful APIs and other sources as first-class citizens in the templating language.

#### Overview

Bindings are specified in the `bindings` array in the Front Matter of layouts and template files.  Here is a basic example that will perform an HTTP GET against a URL, parse the output, and store the parsed results in a variable that can be used anywhere inside the template.

```
---
bindings:
-   name:     todos
    resource: https://jsonplaceholder.typicode.com/todos/
---
<h1>TODO List</h1>
<ul>
{{ range $todo := $.bindings.todos }}
    <li
        {{ if $todo.completed }}
        style="text-decoration: line-through;"
        {{ end }}
    >
        {{ $todo.title }}
    </li>
{{ end }}
</ul>
```

#### Controlling the Request

The `name` and `resource` properties are required for a binding to run, but there are many other optional values supported that allow you to control how the request is performed, how the response if parsed (if at all), as well as what to do if an error occurs (e.g.: connection errors, timeouts, non-2xx HTTP statuses).  These properties are as follows:

| Property Name          | Acceptable Values             | Default Value | Description
| ---------------------- | ----------------------------- | ------------- | -----------
| `body`                 | Object                        | -             |
| `fallback`             | Anything                      | -             |
| `formatter`            | `json`, `form`                | `json`        | Specify how the `body` should be serialized before performing the request.
| `if_status`            | Anything                      | -             | Actions to take when specific HTTP response codes are encountered.
| `insecure`             | `true`, `false`               | `false`       | Whether SSL/TLS peer verification should be enforced.
| `method`               | String                        | `get`         | The HTTP method to use when making the request.
| `no_template`          | `true`, `false`               | `false`       |
| `not_if`               | String                        | -             | If this value or expression yields a truthy value, the binding will not be evaluated.
| `on_error`             | String                        | -             | What to do if the request fails.
| `only_if`              | String                        | -             | Only evaluate if this value or expression yields a truthy value.
| `optional`             | `true`, `false`               | `false`       | Whether a response error causes the entire template render to fail.
| `param_joiner`         | String                        | `;`           | When a key in `params` is specified as an array, how should those array elements be joined into a single string value.
| `params`               | Object                        | -             | An object representing the query string parameters to append to the URL in `resource`.  Keys may be any scalar value or array of scalar values.
| `parser`               | `json`, `html`, `text`, `raw` | `json`        | Specify how the response body should be parsed into the binding variable.
| `rawbody`              | String                        | -             | The *exact* string to send as the request body.
| `skip_inherit_headers` | `true`, `false`               | `false`       | If true, no headers from the originating request to render the template will be included in this request, even if Header Passthrough is enabled.

#### Handling Response Codes and Errors



#### Conditional Evaluation
#### Repeaters

### Functions
#### ... all the functions ...

### Dynamic Variables

Sometimes it is useful to be able to dynamically manipulate data during template evaluation.  Diecast has a set of functions that allow for custom data to be set, retrieved, and removed at runtime.

#### Runtime Variable Functions

##### `var "VARNAME" [VALUE]`

The `var` function declares a new variable with a given name, optionally setting it to an initial value.  If a value is not provided, the variable is set to a `nil` value.  This is also how you can clear out the value of an existing variable.

The string defining the variable name is interpreted as a "dot.separated.path" that is used to set the value in a deeply-nested object.  For example, the following code:

```
var "user.auth.scheme" "basic"
```

...would produce the following struct:

```
{
    "vars": {
        "user": {
            "auth": {
                "scheme": "basic"
            }
        }
    }
}
```

...and would be accessible with the code `{{ $.vars.user.auth.scheme }}`

##### `push "VARNAME" VALUE`

The `push` function appends the given _VALUE_ to the variable at _"VARNAME"_.  If the current value is nil, the result will be an array containing just the element `[ VALUE ]`.  If the current value is not an array, it will first be converted to one.  Then _VALUE_ will be appended to the array.

##### `pop "VARNAME"`

The `pop` function remove the last element from the array at variable _"VARNAME"_.  This value will be returned, or if the array is non-existent or empty, will return `nil`.

### Postprocessors

Postprocessors are routines that are run after the template is rendered for a request, but before the response is returned to the client.  This allows for actions to be taken on the final output, processing it in various ways.

#### Prettify HTML

The `prettify-html` postprocessor will treat the incoming document as HTML, running it through an autoformatter and autoindent routine.  This is useful for ensuring that well-formed and visually pleasing HTML is emitted from Diecast.

#### Trim Empty Lines

The `trim-empty-lines` postprocessor removes all lines from the final document that are zero-length or only contain whitespace.  This is especially useful when producing responses encoded as Tab-Separated Values (TSV) or Comma-Separated Values (CSV).



### Renderers

Diecast supports various methods of converting the output of the rendered templates and layouts into a finished product that can be delvered to the client.  Renderers receive the rendered template as input and are responsible for writing _something_ to the client.

#### HTML

The HTML renderer ensures that external template content, that is, template data sourced from a variable or function, is escaped properly within the context of the type of file being processed (HTML, CSS, or Javascript.)  This makes user-defined content safe to use in Diecast because it will always be sanitized before being returned to the client.  The `html` renderer is the default renderer if no other renderer is specified.

#### PDF

The `pdf` renderer is used in tandem with the HTML renderer to convert the HTML page into a PDF document that is then returned to the client.

#### Sass

The `sass` renderer takes file or template output and compiles it on the fly using the `libsass` library.  This is the default renderer for files matching the pattern `*.scss`.

#### [ Image / PNG / JPG / GIF ]

### Mounts
#### File
#### HTTP
