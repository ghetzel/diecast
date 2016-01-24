# diecast [![GoDoc](https://godoc.org/github.com/ghetzel/diecast?status.svg)](https://godoc.org/github.com/ghetzel/diecast)

A dynamic site generator that consumes REST services and renders static HTML output in realtime

# Overview

`diecast` is a utility and importable Golang package that allows you to dynamically render a directory tree of template files into HTML.  Data can be retrieved from third-party API sources and on-the-fly included during the template rendering process.  This allows you to create entire websites that consume external data sources and present complete UI from them.

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

# Template Directory

Templates are dynamically rendered upon request. The route structure of the site matches that of the template directory.  For example, if the `templates` directory contains a file called `search.pongo`, you can access that page at [http://localhost:28419/search].  If there is another file at `orders/index.pongo`, it will be accessible at [http://localhost:28419/orders].