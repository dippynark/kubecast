module Http.Decorators exposing
    ( RawRequest
    , defaultPost, defaultGet, defaultGetString
    , sendRaw, toTaskRaw, toRequest
    , cacheBusterUrl, addCacheBuster, taskWithCacheBuster, sendWithCacheBuster
    )

{-| This module contains several functions that build on the
[`elm-lang/http`](/packages/elm-lang/http/1.0.0) module.
Note that `interpretStatus` and `promoteError` are no longer included, because
the `Http` module now does what they used to do.

## Transparent Requests

@docs RawRequest, defaultPost, defaultGet, defaultGetString, sendRaw, toTaskRaw, toRequest

## Cache busting

@docs cacheBusterUrl, addCacheBuster, taskWithCacheBuster, sendWithCacheBuster

-}

import Http exposing (Request, Error, Header, Body, Expect, expectJson, expectString, emptyBody)
import Task exposing (Task)
import Time exposing (Time)
import String exposing (contains, endsWith)
import Json.Decode


{-| The [`Request`](/packages/elm-lang/http/1.0.0/Http#Request) type in the `Http` module
is opaque, in the sense that once you have a `Request`, you cannot extract its
parts in order to construct a different `Request`.  The `RawRequest` type is a
workaround for that, allowing you to work with the parts of a request.

You can construct a `RawRequest` manually, by filling in its parts. The various parts
have the same meaning as the parameters to [`Http.request`](/packages/elm-lang/http/1.0.0/Http#request).

    req : RawRequest String
    req =
        { method = "GET"
        , headers = [header "X-Test-Header" "Foo"]
        , url = "http://elm-lang.org"
        , body = Http.emptyBody
        , expect = Http.expectString
        , timeout = Nothing
        , withCredentials = False
        }

Alternatively, you can construct a `RawRequest` by using [`defaultGet`](#defaultGet),
[`defaultGetString`](#defaultGetString), or [`defaultPost`](#defaultPost) to fill in
some defaults. These functions are like [`Http.get`](/packages/elm-lang/http/1.0.0/Http#get),
[`Http.getString`](/packages/elm-lang/http/1.0.0/Http#getString) and
[`Http.post`](/packages/elm-lang/http/1.0.0/Http#post), except that they return a
`RawRequest` which you can further customize, rather than an opaque
[`Request`](/packages/elm-lang/http/1.0.0/Http#Request).

    -- This produces a `RawRequest` equivalent to the manually-constructed
    -- example above.
    req : RawRequest String
    req =
        let default = defaultGetString "http://elm-lang.org"
        in {default | headers = [header "X-Test-Header" "Foo"]}

Once you have a `RawRequest`, you can use [`toRequest`](#toRequest) to turn it into a
[`Request`](/packages/elm-lang/http/1.0.0/Http#Request) that the `Http` module can use.
Alternatively, you can [`sendRaw`](#sendRaw) or
[`toTaskRaw`](#toTaskRaw) to turn the `RawRequest` directly into a `Cmd` or `Task`.
-}
type alias RawRequest a =
    { method : String
    , headers : List Header
    , url : String
    , body : Body
    , expect : Expect a
    , timeout : Maybe Time
    , withCredentials : Bool
    }


{-| Like [`Http.getString`](/packages/elm-lang/http/1.0.0/Http#getString), but returns
a `RawRequest String` that you could further customize.

You can then use [`toRequest`](#toRequest) to make an actual
[`Http.Request`](/packages/elm-lang/http/1.0.0/Http#Request), or supply
the `RawRequest` to [`sendRaw`](#sendRaw) or [`toTaskRaw`](#toTaskRaw).

    req : RawRequest String
    req =
        let default = defaultGet "http://elm-lang.org"
        in {default | timeout = Just (1 * Time.second)}
-}
defaultGetString : String -> RawRequest String
defaultGetString url =
    { method = "GET"
    , headers = []
    , url = url
    , body = emptyBody
    , expect = expectString
    , timeout = Nothing
    , withCredentials = False
    }


{-| Like [`Http.get`](/packages/elm-lang/http/1.0.0/Http#get), but returns a
`RawRequest String` that you could further customize.

You can then use [`toRequest`](#toRequest) to make an actual
[`Http.Request`](/packages/elm-lang/http/1.0.0/Http#Request), or supply
the `RawRequest` to [`sendRaw`](#sendRaw) or [`toTaskRaw`](#toTaskRaw).
-}
defaultGet : String -> Json.Decode.Decoder a -> RawRequest a
defaultGet url decoder =
    { method = "GET"
    , headers = []
    , url = url
    , body = emptyBody
    , expect = expectJson decoder
    , timeout = Nothing
    , withCredentials = False
    }


{-| Like [`Http.post`](/packages/elm-lang/http/1.0.0/Http#post), but returns
a `RawRequest` that you could further customize.

You can then use [`toRequest`](#toRequest) to make an actual
[`Http.Request`](/packages/elm-lang/http/1.0.0/Http#Request), or supply
the `RawRequest` to [`sendRaw`](#sendRaw) or [`toTaskRaw`](#toTaskRaw).
-}
defaultPost : String -> Body -> Json.Decode.Decoder a -> RawRequest a
defaultPost url body decoder =
    { method = "POST"
    , headers = []
    , url = url
    , body = body
    , expect = expectJson decoder
    , timeout = Nothing
    , withCredentials = False
    }


{-| Turns a `RawRequest a` into an
[`Http.Request a`](/packages/elm-lang/http/1.0.0/Http#Request).
This is just another name for
[`Http.request`](/packages/elm-lang/http/1.0.0/Http#request).
-}
toRequest : RawRequest a -> Http.Request a
toRequest = Http.request


{-| Like [`Http.send`](/packages/elm-lang/http/1.0.0/Http#send),
but uses a `RawRequest` instead of a
[`Request`](/packages/elm-lang/http/1.0.0/Http#Request).
-}
sendRaw : (Result Http.Error a -> msg) -> RawRequest a -> Cmd msg
sendRaw tagger req =
    Http.send tagger (toRequest req)


{-| Like [`Http.toTask`](/packages/elm-lang/http/1.0.0/Http#toTask),
but uses a `RawRequest` instead of a
[`Request`](/packages/elm-lang/http/1.0.0/Http#Request).
-}
toTaskRaw : RawRequest a -> Task Http.Error a
toTaskRaw =
    Http.toTask << toRequest


{-| Given a URL, add a 'cache busting' parameter of the form
'?cacheBuster=219384729384', where the number is derived from the current time.
The purpose of doing this would be to help defeat any caching that might
otherwise take place at some point between the client and server.

Often you won't need to call this directly -- you can use [`addCacheBuster`](#addCacheBuster),
[`taskWithCacheBuster`](#taskWithCacheBuster) or [`sendWithCacheBuster`](#sendWithCacheBuster) instead.

    -- Should resolve to something like "http://elm-lang.org?cacheBuster=12348257"
    urlWithTime : Task x String
    urlWithTime =
        cacheBusterUrl "http://elm-lang.org"

    -- Should resolve to something like "http://elm-lang.org?param=7&cacheBuster=12348257"
    urlWithTime2 : Task x String
    urlWithTime2 =
        cacheBusterUrl "http://elm-lang.org?param=7"

-}
cacheBusterUrl : String -> Task x String
cacheBusterUrl url =
    let
        -- essentially, we want to add ?cacheBuster=123482
        -- or, &cacheBuster=123482
        urlWithTime time =
            urlWithParamSeparator ++ "cacheBuster=" ++ (toString time)

        urlWithParamSeparator =
            if endsWith "?" urlWithQueryIndicator
               then urlWithQueryIndicator
               else urlWithQueryIndicator ++ "&"

        urlWithQueryIndicator =
            if contains "?" url
                then url
                else url ++ "?"

    in
        Task.map urlWithTime Time.now


{-| Given a `RawRequest`, add a cache-busting parameter to the URL. This uses
[`cacheBusterUrl`](#cacheBusterUrl) internally to generate the parameter.

Note that the resulting task will resolve with the modified `RawRequest`, which
will itself need to be turned into a `Task` or `Cmd` to be actually executed.
Thus, you often would not need to call this directly -- you could use
[`taskWithCacheBuster`](#taskWithCacheBuster) or [`sendWithCacheBuster`](#sendWithCacheBuster)
instead. You would only need `addCacheBuster` in cases where you need to do some
further processing of the resolved `RawRequest` before turning it into a `Task` or a `Cmd`.
-}
addCacheBuster : RawRequest a -> Task x (RawRequest a)
addCacheBuster req =
    Task.map
        (\s -> {req | url = s})
        (cacheBusterUrl req.url)


{-| Given a `RawRequest`, add a cache-busting parameter to the URL and return
a `Task` that executes the request.

This is useful in cases where the resulting `Task` is part of some larger
chain of tasks. In cases where you are just going to turn this very `Task`
into a `Cmd`, you could use [`sendWithCacheBuster`](#sendWithCacheBuster) instead.
-}
taskWithCacheBuster : RawRequest a -> Task Error a
taskWithCacheBuster req =
    Task.andThen toTaskRaw (addCacheBuster req)


{-| Given a `RawRequest`, add a cache-busting parameter to the URL and return
a `Cmd` that executes the request.

This is a convenience for cases in which your `RawRequest` is meant to result
in a simple `Cmd`. For more complex cases, you can use [`taskWithCacheBuster`](#taskWithCacheBuster),
[`addCacheBuster`](#addCacheBuster) or [`cacheBusterUrl`](#cacheBusterUrl), depending
on how much further customization you need.
-}
sendWithCacheBuster : (Result Error a -> msg) -> RawRequest a -> Cmd msg
sendWithCacheBuster tagger req =
    Task.attempt tagger (taskWithCacheBuster req)
