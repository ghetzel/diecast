# diecast [![GoDoc](https://godoc.org/github.com/ghetzel/diecast?status.svg)](https://godoc.org/github.com/ghetzel/diecast)

A dynamic site generator that consumes REST services and renders static HTML output in realtime

# Overview

`diecast` is a utility and importable Golang package that allows you to dynamically render a directory tree of template files into HTML.  Data can be retrieved from third-party API sources and on-the-fly included during the template rendering process.  This allows you to create entire websites that consume external data sources and present complete UI from them.
