# diecast [![GoDoc](https://godoc.org/github.com/ghetzel/diecast?status.svg)](https://godoc.org/github.com/ghetzel/diecast)

A standalone web server that consumes REST services, passes the response into templates, and serves the results.

# Overview

Diecast is a utility and importable Golang package that allows you to dynamically render a directory tree of template files into HTML,CSS or anything else text-based.  Data can be retrieved from third-party API sources and on-the-fly included during the template rendering process.  This allows you to create entire websites that consume external data sources and present a complete UI for them without the need to write intermediary logic.

# Goals

The primary goal of this project is to make the process of building data-driven web applications easier by reducing cognitive load on the developer.  Rather than tackling the problem using conventional web frameworks, Diecast instead tries to operate more along the lines of a
static site generator (like [Jekyll](https://jekyllrb.com) or [Hugo](https://gohugo.io)), except that it operates as a long-running process that continuously serves and renders templates in real time.  The benefit here (I hope) is that, for relatively small sites that don't require extensive URL routing logic beyond what can be achieved by a filesystem-oriented approach, you gain a declarative interface to consuming data from APIs, using the templates to guide that data along the path to becoming HTML, CSS, Javascript, or any other text-oriented format.

# Example

The following shows an example that illustrates basic usage of `diecast` for a simple "Hello World" site with no external content.  You can try this out by installing Diecast (`go get github.com/ghetzel/diecast`), changing to the `examples/hello-world` directory, and running `diecast`.  The site is available at [http://localhost:28419].

Directory tree:
```
$ cd ./examples/hello-world
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
```

## Navigation

By visiting [http://localhost:28419], you will be presented with a web page representing the contents of the `index.html` file.  Because there is a layout present (in `_layouts/default.html`), the index page will inserted into that layout before being returned.  Using layouts and includes along with templates allows for extensive code reuse throughout your site.

The name of the template file becomes the URL path used to access that file.  For example, the `functions.html` file is accessible by going to [http://localhost:28419/functions].

## Bindings

Bindings are a mechanism that tell Diecast which (if any) remote API resources should be loaded before rendering a template.  In the example above, there are two bindings configured: `current_user` and `objects`.  The `current_user` binding is flagged as _optional_, meaning that if there is an error in the request (connection error, SSL error, non-2xx HTTP status), the value will return `nil` instead of causing a fatal error. The `objects` binding is required, so any errors in retrieval will cause a fatal error in the page.

Bindings, in concert with Templates, are how you consume third-party remote APIs and turn those responses into usable web applications.


## Templating

By default, HTML files served through `diecast` will treated as templates and rendered, with all other files being served as-is (static resources).  You can specify that filenames matching certain
[glob-like](https://golang.org/pkg/path/filepath/#Match) patterns will be treated
as templates and processed using the [rendering engine](https://golang.org/pkg/html/template/).

## Functions

Diecast ships with a suite of built-in functions that can be used to make template development easier.  Check out the
[Diecast Function Reference](FUNCTIONS.md) for more details.

## Layouts

As touched on earlier, it is often desirable for some or all of a site to share a common theme (e.g.: navigation,
headers, scripts).  This can be achieved in Diecast using _layouts_.  Any files in
the `_layouts` directory will be available as wrappers for templates.  If the file `_layouts/default.html` exists,
all templated files will be wrapped in that layout by default with no additional configuration.

A _partial_ is a file whose name starts with an underscore (e.g: `_list.html`).  Partials do *not* get layouts
applied to them automatically, and are designed for making composable and dynamic pages by allowing them to be
included via AJAX calls or the `{{ template }}` statement.