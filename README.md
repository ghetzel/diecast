# diecast [![GoDoc](https://godoc.org/github.com/ghetzel/diecast?status.svg)](https://godoc.org/github.com/ghetzel/diecast)

A dynamic site generator that consumes REST services and renders static HTML output.

# Overview

`diecast` is a utility and importable Golang package that allows you to dynamically render a directory tree of template files into HTML.  Data can be retrieved from third-party API sources and on-the-fly included during the template rendering process.  This allows you to create entire websites that consume external data sources and present a complete UI for them.

# Example

The following shows an example that illustrates basic usage of `diecast` for a simple "Hello World" site with no external content.  You can try this out by installing Diecast (`go get github.com/ghetzel/diecast`), changing to the `examples/hello` directory, and running `diecast serve`.  The site is available at [http://localhost:28419].

Directory tree:
```
$ cd ./examples/hello
$ tree
.
├── public
│   └── image.gif
└── templates
    └── index.pongo

2 directories, 2 files
$ diecast serve
INFO[0000] diecast v0.0.1 started at 2016-01-23 17:39:09.38016741 -0500 EST
INFO[0000] Starting HTTP server at 127.0.0.1:28419
[negroni] listening on 127.0.0.1:28419
```

# Templating

By default, all files served through `diecast` will be transmitted as-is (static
resources).  You can specify that filenames matching certain
[glob-like](https://golang.org/pkg/path/filepath/#Match) patterns will be treated
as templates and processed using the [rendering engine](https://golang.org/pkg/html/template/).

# Layouts

Often it is desirable for some or all of a site to share a common theme (e.g.: navigation,
headers, scripts).  This can be acheived in `diecast` using _layouts_.  Any files in
the `_layouts` directory will be available as wrappers for templates.

# Bindings

Bindings are a mechanism that tell `diecast` which (if any) remote API resources should be loaded before rendering a template.  In the configuration example above, there are two bindings configured: `my_api_endpoint` and `other_api_endpoint`.  When the user requests a route, all bindings that have a pattern in the `routes` array matching the requested route will be evaluated.  Their output will be available to the template as a map-type structure under the top level "data" key.  The results of requesting the `/orders` route in this case would be:

```
{
  "data": {
    "my_api_endpoint": <deserialized response body>,
    "other_api_endpoint": <deserialized response body>
  }
}
```
