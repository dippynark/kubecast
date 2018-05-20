# elm-http-decorators

This package provides some useful types and functions which work with
[elm-lang/http](https://github.com/elm-lang/http)

## Transparent Requests

The `Request` type in the `Http` module
is opaque, in the sense that once you have a `Request`, you cannot extract its
parts in order to construct a different `Request`. 

Thus, we supply a `RawRequest` type (and associated functions) as a
workaround, allowing you to work with the parts of a request.

## Cache Busting

We also provide some functions for adding a cache-busting parameter to
a `RawRequest`.
