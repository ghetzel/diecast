# diecast [![GoDoc](https://godoc.org/github.com/ghetzel/diecast?status.svg)](https://godoc.org/github.com/ghetzel/diecast)

A dynamic site generator that consumes REST services and renders static HTML output.

# Overview

Diecast is a utility and importable Golang package that allows you to dynamically render a directory tree of template files into HTML.  Data can be retrieved from third-party API sources and on-the-fly included during the template rendering process.  This allows you to create entire websites that consume external data sources and present a complete UI for them.

# Example

The following shows an example that illustrates basic usage of `diecast` for a simple "Hello World" site with no external content.  You can try this out by installing Diecast (`go get github.com/ghetzel/diecast`), changing to the `examples/hello` directory, and running `diecast`.  The site is available at [http://localhost:28419].

Directory tree:
```
$ cd ./examples/hello
$ tree
.
├── functions.html
├── home.html
├── image.gif
├── index.html
├── main.css
└── thing
    └── index.html

1 directory, 6 files
$ diecast
2017/11/19 15:37:41 INFO[0001] main: diecast v1.3.1 started at 2017-11-19 15:37:41.351414477 -0500 EST m=+0.003960887
2017/11/19 15:37:41 INFO[0002] main: Starting HTTP server at http://127.0.0.1:28419
```

## A More Complete Example
```
---
bindings:
- name:     current_user
  resource: 'http://my-service.example.com/api/v1/users/me'
  optional: true

- name:     objects
  resource: 'http://my-service.example.com/api/v1/objects'
  params:
    apiKey:  '{{ qs `apiKey` }}' # passthrough the value of the ?apiKey query string in
                                 # the request as a querystring parameter of this resource.

    version: 1                   # this will become 'version=1' in the upstream URL being requested

---
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <link rel="stylesheet" type="text/css" href="/css/main.css">
    <title>Hello!</title>
  </head>

  <body>
    {{ if .bindings.current_user }}
    Welcome, <bold>{{ .bindings.current_user.name }}</bold>!
    {{ else }}
    Welcome, Guest!
    {{ end }}

    <h1>Objects</h1>
    <ul>
    <!-- iterate through each object that came from the "objects" call above. -->
    {{ range .bindings.objects }}
        <!-- ".name" is relative to the current object we're iterating on -->
        <li>{{ .name }}</li>
    {{ end }}
    </ul>
  </body>
</html>
```

# Templating

By default, HTML files served through `diecast` will treated as templates and rendered, with all other files being served as-is (static resources).  You can specify that filenames matching certain
[glob-like](https://golang.org/pkg/path/filepath/#Match) patterns will be treated
as templates and processed using the [rendering engine](https://golang.org/pkg/html/template/).

# Layouts

Often it is desirable for some or all of a site to share a common theme (e.g.: navigation,
headers, scripts).  This can be achieved in Diecast using _layouts_.  Any files in
the `_layouts` directory will be available as wrappers for templates.  If the file `_layouts/default.html` exists,
all templated files will be wrapped in that layout by default with no additional configuration.

A _partial_ is a file whose name starts with an underscore (e.g: `_list.html`).  Partials do *not* get templates
applied to them automatically, and are designed for making composable and dynamic pages by allowing them to be
rendered as templates then included via AJAX calls or via the `{{ template }}` statement.

# Bindings

Bindings are a mechanism that tell Diecast which (if any) remote API resources should be loaded before rendering a template.  In the example above, there are two bindings configured: `current_user` and `objects`.  The `current_user` binding is flagged as _optional_, meaning that if there is an error in the request (connection error, SSL error, non-2xx HTTP status), the value will return `nil` instead of causing a fatal error. The `objects` binding is required, so any errors in retrieval will cause a fatal error in the page.

Bindings, in concert with Templates, are how you consume third-party remote APIs and turn those responses into usable web applications.
