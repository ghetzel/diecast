<div id="logo">
    <img src="/diecast/src/assets/img/diecast-text-light-bg-64.png" alt="Diecast Logo">
</div>


## Introduction

Diecast is a web server that allows you to render a directory tree of template files into HTML, CSS or anything other text-based media in real-time.  You can used Diecast to retrieve data from remote sources during the template rendering process, creating dynamic web pages built by consuming APIs and remote files without the need to use client-side Javascript/AJAX calls or an intermediate server framework.

## Benefits and Useful Features

Diecast is a good choice for any web project that consumes data from a remote data source for the purpose of building web pages or other text-based file types.  Think of Diecast as an intermediary that takes raw data and transforms it, in real-time, into websites.

One of the benefits to this approach is that it enforces separation between data storage and data presentation.  It is often the case that web programming frameworks try to "do it all", they provide access to data, mediate requests, and generate HTML content.  It is easy to create close dependencies between these separate tasks, the kind that can be very hard to change or remove.

Diecast, on the other hand, consumes data primarily via HTTP/RESTful API services.  This means that the service(s) responsible for providing the data can focus on doing that one thing well, while the parts of the application responsible for turning that data into something visual and interactive can focus on doing that.  If you need to change out backend languages in the future, or incorporate new data sources, you can do so without a major overhaul of the frontend components.  The HTTP protocol is language-agnostic.

## Getting Started

Building a site using Diecast is as easy as putting files in a folder.  When you run the `diecast` command in this folder (the _working directory_), Diecast will make the contents of the folder available as a web page (by default, at [`http://localhost:28419`](http://localhost:28419).)  Any file ending in `.html` will be treated as a template and be processed before being returned to the user.  If no other filenames or paths are requested, Diecast will look for and attempt to serve the file `index.html`.

## Installation

<details>
    <summary>Golang / via `go get`</summary>
    <div>
    ```
    go get github.com/ghetzel/diecast/cmd/diecast
    ```
    </div>
</details>
<details>
    <summary>macOS / Homebrew</summary>
    <div></div>
</details>
<details>
    <summary>Windows</summary>
    <div></div>
</details>
<details>
    <summary>Linux</summary>
    <div></div>
</details>
<details>
    <summary>FreeBSD</summary>
    <div></div>
</details>
<details>
    <summary>Binaries</summary>
    <div></div>
</details>
<details>
    <summary>From Source</summary>
    <div></div>
</details>

## URL Structure

Diecast does not have a concept of URL path routing, but rather strives to enforce simple, linear hierarchies by exposing the working directory directly as routable paths.  For example, if a user visits the path `/users/list`, Diecast will look for files to serve in the following order:

* `./users/list/index.html`
* `./users/list.html`
* `./_errors/404.html`
* `./_errors/4xx.html`
* `./_errors/default.html`

The first matching file from the list above will be served.

## Configuration

You can configure Diecast by creating a file called `diecast.yml` in the same folder that the `diecast` command is run in, or by specifying the path to the file with the `--config` command line option.  You can use this configuration file to control how Diecast renders templates and when, as well as set options for how files are accessed and from where.  Diecast tries to use "sane defaults" whenever possible, but you can configure Diecast in many ways to suit your needs.  For more details on these defaults and to see what goes in a `diecast.yml` file, see the [Example Config File](/examples/diecast.sample.yml).

## Templating

Beyond merely acting as a simple file server, Diecast comes with a rich templating environment that you can use to build complex sites in a straightforward, easy to understand way.  Templates are just files that you tell Diecast to treat specially.  The default templating language used by Diecast is [Golang's built-in `text/template` package.](https://golang.org/pkg/text/template/).  Templates files consist of the template content, and optionally a header section called _front matter_.  These headers are used to specify template-specific data such as predefined data structures, paths of other templates to include, rendering options, and the inclusion of remote data via [data bindings](#data-bindings).  An example template looks like this:

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

Golang's `text/template` package provides a fast templating language that is used to generate things like HTML on-the-fly.  When generating HTML, CSS, or Javascript documents, Diecast understands that it is working with these languages and performs automatic escaping of code to ensure the output is safe against many common code injection techniques.  This is especially useful when using templates to render user-defined input that may contain HTML tags or embedded Javascript.

#### Intro to `text/template`

The built-in templating language will be familiar to those coming from a background in other templating languages like [Jekyll](https://jekyllrb.com/), [Jinja2](http://jinja.pocoo.org/docs/2.10/), and [Mustache](https://mustache.github.io/).  Below is a quick guide on how to accomplish certain common tasks when writing templates for Diecast.  For detailed information, check out the [Golang `text/template` Language Overview](https://golang.org/pkg/text/template/#pkg-overview).

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

Diecast defines a global data structure in the `$.page` variable that can be used to provide site-wide values to templates.  You can define the `page` structure in multiple places, which lets you explicitly provide data when serving templates that doesn't come from a data binding.  The `page` structure is inherited by child templates, and all values are merged together to form a single data structure.  For example, given the following files:

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

The final `page` data structure would look like this immediately before processing `index.html`:

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

## Data Bindings

Data Bindings (or just _bindings_) are one of the most important concepts in Diecast.  Bindings are directives you add to the Front Matter of layouts and templates that specify remote URLs to retrieve (via an HTTP client built in to `diecast`), as well as how to handle parsing the response data and what to do about errors.  This concept is extremely powerful, in that it allows you to create complex data-driven sites easily and cleanly by treating remote data from RESTful APIs and other sources as first-class citizens in the templating language.

### Overview

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

### Controlling the Request

The `name` and `resource` properties are required for a binding to run, but there are many other optional values supported that allow you to control how the request is performed, how the response if parsed (if at all), as well as what to do if an error occurs (e.g.: connection errors, timeouts, non-2xx HTTP statuses).  These properties are as follows:

| Property Name          | Acceptable Values             | Default       | Description
| ---------------------- | ----------------------------- | ------------- | -----------
| `body`                 | Object                        | -             |
| `fallback`             | Anything                      | -             |
| `formatter`            | `json, form`                | `json`        | Specify how the `body` should be serialized before performing the request.
| `if_status`            | Anything                      | -             | Actions to take when specific HTTP response codes are encountered.
| `insecure`             | Boolean                       | `false`       | Whether SSL/TLS peer verification should be enforced.
| `method`               | String                        | `get`         | The HTTP method to use when making the request.
| `no_template`          | Boolean                         | `false`       |
| `not_if`               | String                        | -             | If this value or expression yields a truthy value, the binding will not be evaluated.
| `on_error`             | String                        | -             | What to do if the request fails.
| `only_if`              | String                        | -             | Only evaluate if this value or expression yields a truthy value.
| `optional`             | Boolean                       | `false`       | Whether a response error causes the entire template render to fail.
| `param_joiner`         | String                        | `;`           | When a key in `params` is specified as an array, how should those array elements be joined into a single string value.
| `params`               | Object                        | -             | An object representing the query string parameters to append to the URL in `resource`.  Keys may be any scalar value or array of scalar values.
| `parser`               | `json, html, text, raw` | `json`        | Specify how the response body should be parsed into the binding variable.
| `rawbody`              | String                        | -             | The *exact* string to send as the request body.
| `skip_inherit_headers` | Boolean                       | `false`       | If true, no headers from the originating request to render the template will be included in this request, even if Header Passthrough is enabled.

### Handling Response Codes and Errors

By default, the response to a binding's HTTP request must be a [200-series HTTP status code](https://en.wikipedia.org/wiki/List_of_HTTP_status_codes#2xx_Success).  If it is not (e.g. returns an 404 or 500 error), Diecast will return an error page instead.  Custom error pages live in a top-level `_errors` folder.  This folder will be checked for specially-named files that are to be used for handling different error types.  These filenames will be checked, in order, in the event of a binding error:


**`/_errors/404.html`**

Uses the exact HTTP status code that was encountered.  Use this to handle specific, well-known error cases like "404 Not Found" or "500 Internal Server Error".

**`/_errors/4xx.html`**

Specifies an error page that is used for an entire range of error types.  HTTP errors are grouped by type; e.g.: status codes between 400-499 indiciate a _client_ error that the user can fix, whereas 500-599 describes _server_ errors the user cannot do anything about.

**`/_errors/default.html`**

Specifies an error page that is used to handle _any_ non-2xx HTTP status, as well as deeper problems like connection issues, SSL security violations, and DNS lookup problems.


### Conditional Evaluation

By default, all bindings specified in a template are evaluated in the order they appear.  It is sometimes useful to place conditions on whether a binding will evaluate.  You can specify these conditions using the `only_if` and `not_if` properties on a binding.  These properties take a string containing an inline template.  If the template in an `only_if` property returns a "truthy" value (non-empty, non-zero, or "true"), that binding will be run.  Otherwise, it will be skipped.  The inverse is true for `not_if`: if truthy, the binding is not evaluated.


```
---
bindings:
#   --------------------------------------------------------------------------------
#   Always evaluated
-   name:     always
    resource: /api/status
#   --------------------------------------------------------------------------------
#   Only evaluated if the "login" query string is a "truthy" values (i.e.: "1",
#   "true", "yes")
-   name:     user
    resource: /api/users/self
    only_if:  '{{ qs "login" }}'
#   --------------------------------------------------------------------------------
#   Evaluated every day except Mondays
-   name:     i_hate_mondays
    resource: /api/details
    not_if:   '{{ eqx (now "day") "Monday" }}'
---
```

### Repeaters

_TODO_

### Dynamic Variables

Sometimes it is useful to be able to dynamically manipulate data during template evaluation.  Diecast has a set of functions that allow for custom data to be set, retrieved, and removed at runtime.

#### Runtime Variable Functions

##### `var "VARNAME" [VALUE]`

The `var` function declares a new variable with a given name, optionally setting it to an initial value.  If a value is not provided, the variable is set to a `nil` value.  This is also how you can clear out the value of an existing variable.

The string defining the variable name is interpreted as a "dot.separated.path" that is used to set the value in a deeply-nested object.  For example, the following code:

```
var "user.auth.scheme" "basic"
```

...would produce the following object:

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

Diecast supports various methods of converting the output of the rendered templates and layouts into a finished product that can be delivered to the client.  Renderers receive the rendered template as input and are responsible for writing _something_ to the client.

#### HTML

The HTML renderer ensures that external template content, that is, template data sourced from a variable or function, is escaped properly within the context of the type of file being processed (HTML, CSS, or Javascript.)  This makes user-defined content safe to use in Diecast because it will always be sanitized before being returned to the client.  The `html` renderer is the default renderer if no other renderer is specified.

#### PDF

The `pdf` renderer is used in tandem with the HTML renderer to convert the HTML page into a PDF document that is then returned to the client.

#### Sass

The `sass` renderer takes file or template output and compiles it on the fly using the `libsass` library.  This is the default renderer for files matching the pattern `*.scss`.

#### [ Image / PNG / JPG / GIF ]

### Mounts

Another useful feature of Diecast is its ability to expose multiple, overlapping file trees in one logical namespace.  These alternative file trees (called _mounts_) can be located locally or remotely, and through careful configuration of the scope and order of mounts, fairly complex serving configurations can be achieved.

For example, lets take a GET request to the path `/assets/js/my-lib.js`.  By default, Diecast will look for this path relative to the working directory, and if the file is not found, will return an error.  However, a mount can be configured to handle this path (or its parent folder(s)), serving the file from another directory outside of the working directory, or from another server entirely.

Mounts can also be stacked, in which the URL path they handle refers to multiple possible locations.  When multiple mounts are eligible to handle a request, the requested file is passed to each mount in the order they are defined.  The first mount to successfully handle the file will do so.  This setup can be used to present multiple directories as a single logical one, as well as providing useful fallbacks and proxying capabilities granular to the individual file level.

#### File

The file mount type is used for mount sources that do not begin with a URL scheme.  This means paths like `/usr/share/www/` or `./some/other/path`.  Consider the following mount configuration:

```yaml
mounts:
-   mount: /usr/share/www/
    to:    /assets/
```

A request for the file `/assets/js/my-lib.js` would first attempt to find that file at the path `/usr/share/www/js/my-lib.js`.  Note that the `/assets/` part of the URL path was substituted for the value of the `mount` key.

#### HTTP

The HTTP (aka _proxy_) mount type is used for sources starting with `http://` or `https://`.  In this configuration, the behavior matches that of the File type, except the content is sourced by making an HTTP request to a URL.  Consider the following mount configuration:

```yaml
mounts:
-   mount: https://assets.example.com/
    to:    /assets/
```

A request for the file `/assets/js/my-lib.js` here would result in an HTTP GET request to `https://assets.example.com/assets/js/my-lib.js`.  If the response is a 2xx-series status code, the response body will be sent to the client as if the file resided on the server itself.  Note that in this case, the entire original URL path is sent along to the remote server.

This is useful because it allows Diecast to act as a proxy server, while still layering on features like additional mounts and authenticators.  This means Diecast can be configured to proxy a website, but intercept and substitute requests for specific files on a case-by-case basis.  For example:

```yaml
mounts:
-   mount: /usr/share/custom-google-logos/
    to:    /logos/

-   mount: https://google.com
    to:    /
```

In this configuration, a request to `http://localhost:28419/` will load Google's homepage, but when the browser attempts to load the logo (typically located at `/logos/...`), _that_ request will be routed to the local `/usr/share/custom-google-logos/` directory.  So if the logo for that day is at `/logos/doodles/2018/something.png`, and the file `/usr/share/custom-google-logos/doodles/2018/something.png` exists, that file will be served in lieu of the version on Google's servers.