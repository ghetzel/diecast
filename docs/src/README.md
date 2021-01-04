<div id="logo">
    <img src="/diecast/src/assets/img/diecast-text-light-bg-64.png" alt="Diecast Logo">
</div>

## Introduction

Diecast is a web server that allows you to render a directory tree of template files into HTML, CSS or anything other text-based media in real-time. You can used Diecast to retrieve data from remote sources during the template rendering process, creating dynamic web pages built by consuming APIs and remote files without the need to use client-side Javascript/AJAX calls or an intermediate server framework.

## Benefits and Useful Features

Diecast is a good choice for any web project that consumes data from a remote data source for the purpose of building web pages or other text-based file types. Think of Diecast as an intermediary that takes raw data and transforms it, in real-time, into websites.

One of the benefits to this approach is that it enforces separation between data storage and data presentation. It is often the case that web programming frameworks try to "do it all", they provide access to data, mediate requests, and generate HTML content. It is easy to create close dependencies between these separate tasks, the kind that can be very hard to change or remove.

Diecast, on the other hand, consumes data primarily via HTTP/RESTful API services. This means that the service(s) responsible for providing the data can focus on doing that one thing well, while the parts of the application responsible for turning that data into something visual and interactive can focus on doing that. If you need to change out backend languages in the future, or incorporate new data sources, you can do so without a major overhaul of the frontend components. The HTTP protocol is language-agnostic.

## Getting Started

Building a site using Diecast is as easy as putting files in a folder. When you run the `diecast` command in this folder (the _working directory_), Diecast will make the contents of the folder available as a web page (by default, at [`http://localhost:28419`](http://localhost:28419).) Any file ending in `.html` will be treated as a template and be processed before being returned to the user. If no other filenames or paths are requested, Diecast will look for and attempt to serve the file `index.html`.

## Installation

TODO: make this section much more detailed.

```
GO111MODULE=on go get github.com/ghetzel/diecast/cmd/diecast
```

<!--
<details>
    <summary>Golang / via `go get`</summary>
    <div>

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
-->

## URL Structure

Diecast does not have a concept of URL path routing, but rather strives to enforce simple, linear hierarchies by exposing the working directory directly as routable paths. For example, if a user visits the path `/users/list`, Diecast will look for files to serve in the following order:

- `./users/list/index.html`
- `./users/list.html`
- `./_errors/404.html`
- `./_errors/4xx.html`
- `./_errors/default.html`

The first matching file from the list above will be served.

## Configuration

You can configure Diecast by creating a file called `diecast.yml` in the same folder that the `diecast` command is run in, or by specifying the path to the file with the `--config` command line option. You can use this configuration file to control how Diecast renders templates and when, as well as set options for how files are accessed and from where. Diecast tries to use "sane defaults" whenever possible, but you can configure Diecast in many ways to suit your needs. For more details on these defaults and to see what goes in a `diecast.yml` file, see the [Example Config File](https://github.com/ghetzel/diecast/blob/master/examples/diecast.sample.yml).

## Templating

Beyond merely acting as a simple file server, Diecast comes with a rich templating environment that you can use to build complex sites in a straightforward, easy to understand way. Templates are just files that you tell Diecast to treat specially. The default templating language used by Diecast is [Golang's built-in `text/template` package.](https://golang.org/pkg/text/template/). Templates files consist of the template content, and optionally a header section called _front matter_. These headers are used to specify template-specific data such as predefined data structures, paths of other templates to include, rendering options, and the inclusion of remote data via [data bindings](#data-bindings). An example template looks like this:

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

Golang's `text/template` package provides a fast templating language that is used to generate things like HTML on-the-fly. When generating HTML, CSS, or Javascript documents, Diecast understands that it is working with these languages and performs automatic escaping of code to ensure the output is safe against many common code injection techniques. This is especially useful when using templates to render user-defined input that may contain HTML tags or embedded Javascript.

#### Intro to `text/template`

The built-in templating language will be familiar to those coming from a background in other templating languages like [Jekyll](https://jekyllrb.com/), [Jinja2](http://jinja.pocoo.org/docs/2.10/), and [Mustache](https://mustache.github.io/). Below is a quick guide on how to accomplish certain common tasks when writing templates for Diecast. For detailed information, check out the [Golang `text/template` Language Overview](https://golang.org/pkg/text/template/#pkg-overview).

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

In addition to rendering individual files as standalone pages, Diecast also supports layouts. Layouts serve as wrapper templates for the files being rendered in a directory tree. Their primary purpose is to eliminate copying and boilerplate code. Layouts are stored in a top-level directory called `_layouts`. If the layout `_layouts/default.html` is present, it will automatically be used by default (e.g.: without explicitly specifying it in the Front Matter) on all pages. The layout for any page can be specified in the `layout` Front Matter property, and the special value `layout: none` will disable layouts entirely for that page.

### Page Object

Diecast defines a global data structure in the `$.page` variable that can be used to provide site-wide values to templates. You can define the `page` structure in multiple places, which lets you explicitly provide data when serving templates that doesn't come from a data binding. The `page` structure is inherited by child templates, and all values are merged together to form a single data structure. For example, given the following files:

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

## Conditional Template Loading (Switches)

Diecast templates have a feature that allows you to conditionally switch to loading different templates based on conditions you specify. This is extremely useful for things like loading a different homepage for logged-in users vs. logged out ones.

```
switch:
    - condition: '{{ not (isEmpty $.bindings.current_user) }}'
      use: "/logged-in-homepage.html"

    - use: "/logged-out-homepage.html"
```

### Switch Types

Switch conditions can perform various checks, which interpret the `condition` value differently depending on the check type.

| Value of `type`             | What `condition` should be                                  |
| --------------------------- | ----------------------------------------------------------- |
| `""` or `"expression"`      | A valid template expression that yields a `true` value      |
| `"qs:name_of_query_string"` | A string value for the querystring `name_of_query_string`   |
| `"header:x-my-header"`      | A string value for the header `x-my-header` / `X-My-Header` |

## Data Bindings

Data Bindings (or just _bindings_) are one of the most important concepts in Diecast. Bindings are directives you add to the Front Matter of layouts and templates that specify remote URLs to retrieve (via an HTTP client built in to `diecast`), as well as how to handle parsing the response data and what to do about errors. This concept is extremely powerful, in that it allows you to create complex data-driven sites easily and cleanly by treating remote data from RESTful APIs and other sources as first-class citizens in the templating language.

### Overview

Bindings are specified in the `bindings` array in the Front Matter of layouts and template files. Here is a basic example that will perform an HTTP GET against a URL, parse the output, and store the parsed results in a variable that can be used anywhere inside the template.

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

The `name` and `resource` properties are required for a binding to run, but there are many other optional values supported that allow you to control how the request is performed, how the response if parsed (if at all), as well as what to do if an error occurs (e.g.: connection errors, timeouts, non-2xx HTTP statuses). These properties are as follows:

| Property Name          | Acceptable Values               | Default | Description                                                                                                                                                                                          |
| ---------------------- | ------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`                 | String                          | -       | The name of the variable (under `$.bindings`) where the binding's data is stored.                                                                                                                    |
| `resource`             | String                          | -       | The URL to retrieve. This can be a complete URL (e.g.: "https://...") or a relative path. If a path is specified, the value [bindingPrefix] will be prepended to the path before making the request. |
| `body`                 | Object                          | -       | An object that will be encoded according to the value of `formatter` and used as the request body.                                                                                                   |
| `except`               | []String                        | -       | If set, and the request path matches _any_ of the paths/path globs herein, the binding will not evaluate and be marked optional.                                                                     |
| `fallback`             | Anything                        | -       | If the binding is optional and returns a non-2xx status, this value will be used instead of `null`.                                                                                                  |
| `formatter`            | `json, form`                    | `json`  | Specify how the `body` should be serialized before performing the request.                                                                                                                           |
| `headers`              | Object                          | -       | An object container HTTP request headers to be included in the request.                                                                                                                              |
| `if_status`            | Anything                        | -       | Actions to take when specific HTTP response codes are encountered.                                                                                                                                   |
| `insecure`             | Boolean                         | `false` | Whether SSL/TLS peer verification should be enforced.                                                                                                                                                |
| `method`               | String                          | `get`   | The HTTP method to use when making the request.                                                                                                                                                      |
| `no_template`          | Boolean                         | `false` | If true, inline expressions in binding values will not be honored.                                                                                                                                   |
| `not_if`               | String                          | -       | If this value or expression yields a truthy value, the binding will not be evaluated.                                                                                                                |
| `on_error`             | String                          | -       | What to do if the request fails.                                                                                                                                                                     |
| `only`                 | []String                        | -       | If set, and the request path does not match _any_ of the paths/path globs herein, the binding will not evaluate and be marked optional.                                                              |
| `only_if`              | String                          | -       | Only evaluate if this value or expression yields a truthy value.                                                                                                                                     |
| `optional`             | Boolean                         | `false` | Whether a response error causes the entire template render to fail.                                                                                                                                  |
| `paginate`             | [Paginate Config])(#pagination) | -       | Paginates through a resultset by performing the binding request repeatedly until an edge condition is met, then returns all the results.                                                             |
| `param_joiner`         | String                          | `;`     | When a key in `params` is specified as an array, how should those array elements be joined into a single string value.                                                                               |
| `params`               | Object                          | -       | An object representing the query string parameters to append to the URL in `resource`. Keys may be any scalar value or array of scalar values.                                                       |
| `parser`               | `json, html, text, raw`         | `json`  | Specify how the response body should be parsed into the binding variable.                                                                                                                            |
| `rawbody`              | String                          | -       | The _exact_ string to send as the request body.                                                                                                                                                      |
| `skip_inherit_headers` | Boolean                         | `false` | If true, no headers from the originating request to render the template will be included in this request, even if Header Passthrough is enabled.                                                     |
| `transform`            | String                          | -       | A [JSONPath](#jsonpath-expressions) expression used to transform the resource response before putting it in `$.bindings`.                                                                            |

### Handling Response Codes and Errors

By default, the response to a binding's HTTP request must be a [200-series HTTP status code](https://en.wikipedia.org/wiki/List_of_HTTP_status_codes#2xx_Success). If it is not (e.g. returns an 404 or 500 error), Diecast will return an error page instead. Custom error pages live in a top-level `_errors` folder. This folder will be checked for specially-named files that are to be used for handling different error types. These filenames will be checked, in order, in the event of a binding error:

**`/_errors/404.html`**

Uses the exact HTTP status code that was encountered. Use this to handle specific, well-known error cases like "404 Not Found" or "500 Internal Server Error".

**`/_errors/4xx.html`**

Specifies an error page that is used for an entire range of error types. HTTP errors are grouped by type; e.g.: status codes between 400-499 indiciate a _client_ error that the user can fix, whereas 500-599 describes _server_ errors the user cannot do anything about.

**`/_errors/default.html`**

Specifies an error page that is used to handle _any_ non-2xx HTTP status, as well as deeper problems like connection issues, SSL security violations, and DNS lookup problems.

### Conditional Evaluation

By default, all bindings specified in a template are evaluated in the order they appear. It is sometimes useful to place conditions on whether a binding will evaluate. You can specify these conditions using the `only_if` and `not_if` properties on a binding. These properties take a string containing an inline template. If the template in an `only_if` property returns a "truthy" value (non-empty, non-zero, or "true"), that binding will be run. Otherwise, it will be skipped. The inverse is true for `not_if`: if truthy, the binding is not evaluated.

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

### Pagination

A binding can be configured to perform its request repeatedly, modifying the binding with data from the previous request, allowing for various API pagination access patterns to be accessed under a single binding.

```
---
bindings:
-   name:     pages
    resource: http://elasticsearch:9200/my_search_index/_search
    params:
        q: 'type:customer'
    paginate:
        total: '{{ $.hits.total }}'       # "total number of results overall"
        count: '{{ count $.hits.hits }}'          # "number of results on this page"
        done:  '{{ eqx (count $.hits.hits) 0 }}'  # this should evaluate to a truthy value when we're out hits
        max:   10000                              # a hard cap on the number of results so we don't paginate forever
        data:  '$.hits.hits'                      # a JSONPath query that will transform the data before putting it in the "data" key in the response
        params:
            size:    '{{ qs "limit" 100 }}'       # set the ?size querystring
            from:    '{{ $.page.counter }}'       # set the ?from querystring to the running results counter
---
```

### JSONPath Expressions

Bindings support a flexible mechanism for transforming response data as read from the binding resource. JSONPath is similar to XPath expressions that allow for data to be selected and filtered from objects and arrays.

- [JSONPath Overview](https://goessner.net/articles/JsonPath/)
- [Expression Tester](https://jsonpath.com/)

## Postprocessors

Postprocessors are routines that are run after the template is rendered for a request, but before the response is returned to the client. This allows for actions to be taken on the final output, processing it in various ways.

### Prettify HTML

The `prettify-html` postprocessor will treat the incoming document as HTML, running it through an autoformatter and autoindent routine. This is useful for ensuring that well-formed and visually pleasing HTML is emitted from Diecast.

### Trim Empty Lines

The `trim-empty-lines` postprocessor removes all lines from the final document that are zero-length or only contain whitespace. This is especially useful when producing responses encoded as Tab-Separated Values (TSV) or Comma-Separated Values (CSV).

## Renderers

Diecast supports various methods of converting the output of the rendered templates and layouts into a finished product that can be delivered to the client. Renderers receive the rendered template as input and are responsible for writing _something_ to the client.

### HTML

The HTML renderer ensures that external template content, that is, template data sourced from a variable or function, is escaped properly within the context of the type of file being processed (HTML, CSS, or Javascript.) This makes user-defined content safe to use in Diecast because it will always be sanitized before being returned to the client. The `html` renderer is the default renderer if no other renderer is specified.

### PDF

The `pdf` renderer is used in tandem with the HTML renderer to convert the HTML page into a PDF document that is then returned to the client.

### Sass

The `sass` renderer takes file or template output and compiles it on the fly using the `libsass` library. This is the default renderer for files matching the pattern `*.scss`.

## Mounts

Another useful feature of Diecast is its ability to expose multiple, overlapping file trees in one logical namespace. These alternative file trees (called _mounts_) can be located locally or remotely, and through careful configuration of the scope and order of mounts, fairly complex serving configurations can be achieved.

For example, lets take a GET request to the path `/assets/js/my-lib.js`. By default, Diecast will look for this path relative to the working directory, and if the file is not found, will return an error. However, a mount can be configured to handle this path (or its parent folder(s)), serving the file from another directory outside of the working directory, or from another server entirely.

Mounts can also be stacked, in which the URL path they handle refers to multiple possible locations. When multiple mounts are eligible to handle a request, the requested file is passed to each mount in the order they are defined. The first mount to successfully handle the file will do so. This setup can be used to present multiple directories as a single logical one, as well as providing useful fallbacks and proxying capabilities granular to the individual file level.

### File

The file mount type is used for mount sources that do not begin with a URL scheme. This means paths like `/usr/share/www/` or `./some/other/path`. Consider the following mount configuration:

```yaml
mounts:
  - mount: /usr/share/www/
    to: /assets/
```

A request for the file `/assets/js/my-lib.js` would first attempt to find that file at the path `/usr/share/www/js/my-lib.js`. Note that the `/assets/` part of the URL path was substituted for the value of the `mount` key.

### HTTP

The HTTP (aka _proxy_) mount type is used for sources starting with `http://` or `https://`. In this configuration, the behavior matches that of the File type, except the content is sourced by making an HTTP request to a URL. Consider the following mount configuration:

```yaml
mounts:
  - mount: https://assets.example.com/
    to: /assets/
```

A request for the file `/assets/js/my-lib.js` here would result in an HTTP GET request to `https://assets.example.com/assets/js/my-lib.js`. If the response is a 2xx-series status code, the response body will be sent to the client as if the file resided on the server itself. Note that in this case, the entire original URL path is sent along to the remote server.

This is useful because it allows Diecast to act as a proxy server, while still layering on features like additional mounts and authenticators. This means Diecast can be configured to proxy a website, but intercept and substitute requests for specific files on a case-by-case basis. For example:

```yaml
mounts:
  - mount: /usr/share/custom-google-logos/
    to: /logos/

  - mount: https://google.com
    to: /
```

In this configuration, a request to `http://localhost:28419/` will load Google's homepage, but when the browser attempts to load the logo (typically located at `/logos/...`), _that_ request will be routed to the local `/usr/share/custom-google-logos/` directory. So if the logo for that day is at `/logos/doodles/2018/something.png`, and the file `/usr/share/custom-google-logos/doodles/2018/something.png` exists, that file will be served in lieu of the version on Google's servers.

## Authenticators

Diecast exposes the capability to add authentication and authorization to your applications through the use of configurable _authenticators_. These are added to the `diecast.yml` configuration file, and provide a very flexible mechanism for protecting parts or all of the application using a variety of backends for verifying users and user access.

### Example: Basic HTTP Authentication for the Whole Site

Here is a configuration example that shows how to prompt a client for a username and password when accessing any part of the current site. The permitted users are stored in a standard [Apache `.htaccess` file](http://www.htaccess-guide.com/), which is consulted every time a requested path matches. If specific path patterns aren't included or excluded, all paths will be protected.

```
authenticators:
-   type: basic
    options:
        realm:    'Da Secure Zone'
        htpasswd: '/etc/my-app/htpasswd'
```

### Including and Excluding Paths

It is possible to specify specific paths (or [wildcard patterns](#wildcard-patterns)) for which the authenticator must or must not be applicable to.

To specify a specific set of paths, use the `path` configuration option:

```
authenticators:
-   type: basic
    paths:
    - '/very/specific/secret/place'
    - `/secure/**`
```

To specify a set of paths to exclude from authentication, use the `exclude` option:

```
authenticators:
-   type: basic
    exclude:
    - '/assets/**'
```

### Authenticator Types

Several authentication/authorization backends are supported.

### `type: "basic"`

Prompts users for credentials using [Basic access HTTP Authentication](https://en.wikipedia.org/wiki/Basic_access_authentication), a widely-supported authentication mechanism supported by most web browsers and command-line tools.

#### Supported Options

| Option        | Description                                                                                                                                  |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `realm`       | The Realm exposed with Basic HTTP Authentication; specifies parts of the site that may share credentials.                                    |
| `htpasswd`    | A server-side file path to an [Apache htaccess file](http://www.htaccess-guide.com/) that contains valid usernames and password hashes.      |
| `credentials` | Similar to `htpasswd`, this option is a map that allows you to place `username: 'password-hash'` pairs directly into the configuration file. |

### `type: "oauth2"`

Allows for third-party authentication providers (Google, Facebook, GitHub, etc.) to be used for authenticating a user session. This authenticator requires the `callback` configuration option, which specifies a complete URL that the third-party will send users to upon successful login using their service.

#### Supported Options

| Option        | Description                                                                                                                                                                |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `provider`    | The name of a built-in OAuth2 provider, or "custom". Built-in providers include: `amazon`, `facebook`, `github`, `gitlab`, `google`, `microsoft-live`, `slack`, `spotify`. |
| `client_id`   | The Client ID provided by your OAuth2 provider.                                                                                                                            |
| `secret`      | The Client Secret provided by your OAuth2 provider.                                                                                                                        |
| `scopes`      | A list of named scopes requested from the OAuth2 provider.                                                                                                                 |
| `cookie_name` | The name of the session cookie stored in the user's browser.                                                                                                               |
| `lifetime`    | How long the authenticated session will last.                                                                                                                              |
| `auth_url`    | If provider is "custom", specifies the OAuth2 authentication URL.                                                                                                          |
| `token_url`   | If provider is "custom", specifies the OAuth2 validation URL.                                                                                                              |

## Actions

In addition to serving file and processing templates, Diecast also includes support for performing basic server-side actions. These actions are exposed and triggered by a RESTful web API that is implemented in the `diecast.yml` configuration file. The data made available through these custom API endpoints is gathered by executing shell commands server-side, and as such comes with certain innate risks that need to be addressed in order to maintain a secure application environment.

### WARNING: Danger Zone

Be advised that this feature can _very easily_ be misused or implemented in an insecure way. This feature is intended to provide extremely basic reactive capabilities in an otherwise constrained deployment environment. Diecast will take some measures to sanitize user inputs before inserting them into shell commands, and will not allow the program to even start as the `root` user if actions are defined (unless the `DIECAST_ALLOW_ROOT_ACTIONS` environment variable is set to "true"). However, none of this is a substitute for strictly controlling the scope and use of this feature.

Some tips for making the most effective use of this feature:

- **Use sparingly.** The most secure code is the code that is never written.
- **Use [Authenticators](#authenticators).** Protect your action endpoints with some form of authentication.
- **Keep it simple.** If you find your logic getting too complex or too clever, re-evaluate your needs and approach.

Be careful out there.

### Paths and Steps

Actions are defined under the `actions` configuration key, and are an array of definitions that specify a URL path that, when requested, will execute a sequence of steps and respond with the resulting output. In this way, data can be retrieved, manipulated, and returned to the user. Below is a very basic example that will return the server's current time:

```
actions:
- path:   /api/time
  method: get
  steps:
  - type:   shell
    parser: lines
    data:   date
```

The output a user would see if they visit http://localhost:28419/api/time would look like this:

```
["Tue Jan 02 15:04:05 MST 2006"]
```

Steps are defined as an array of specific actions, be it executing a shell script, converting output from one format to another, or sorting and filtering data. Steps are designed to be chained together to create composable data processing pipelines.

#### Step Type `shell`

The shell step is used to execute a server-side program and return the output. A successful program will run, optionally using the environment variables provided to make use of request-specific details, and print the results to standard output (by default, a JSON-encoded document, but this can be controlled with the `parser` option). The program should exit with a status of zero (0) to indicate success, and non-zero to indicate failure. If the program exits with a non-zero status, the standard output is assumed to be a descriptive error.

##### Shell Environment

Below are the environment variables made available to the `shell` step on invocation:

- `REQ_HEADER_*`: Represents all HTTP headers supplied by the user, prefixed with `REQ_HEADER_`, with the header name upper-cased and all non-alphanumeric characters converted to an underscore (`_`).

- `REQ_PARAM_*`: Represents all query string parameters on the request URL, also upper-cased and underscore-separated.

  - Example: `/api/test?is_enabled=true&hello=there` will yield two environment variables:
    - `REQ_PARAM_IS_ENABLED=true`
    - `REQ_PARAM_HELLO=there`

- `REQ_PARAM_*`: Represents positional parameters in the URL, specified in the `path` configuration in the action. For example, if `path: '/api/actions/:action-name'`, then the script will be called with the environment variable `REQ_PARAM_ACTION_NAME`. If both a URL parameter and query string parameter have the same name, the URL parameter will overwrite the query string parameter.

#### Step Type `process`

The process step is used to manipulate the output from a previous step in some way. This can be used to convert script output into complex nested data structures, sort lines of text, or perform other operations on the data.

##### Process Operation

What action to take on input data is specified in the step's `data` option.

| Process Operation | Description                                                                                                                                                                                                    |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `sort`            | Sort an array of strings lexically ascending order.                                                                                                                                                            |
| `rsort`           | Same as `sort`, but in descending order.                                                                                                                                                                       |
| `diffuse`         | Takes an array of strings in the format "nested.key.name=value" and converts that into a deeply-nested map (e.g.: `{"nested": {"key": {"name": "value"}}}`). Data types for values are automatically detected. |

##### Examples

Creates an endpoint at [http://localhost:28419/api/deploy] that returns an object containing the current git branch and revision of the
Diecast runtime directory:

```
actions:
-   path:   /api/deploy
    method: get
    steps:
    -   type: shell
        data: |
            #!/usr/bin/env bash
            echo "branch=$(git rev-parse --abbrev-ref HEAD)"
            echo "revision=$(git rev-parse HEAD)"

    -   type: process
        data: diffuse
```
