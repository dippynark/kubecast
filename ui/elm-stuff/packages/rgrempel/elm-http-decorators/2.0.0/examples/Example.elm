
import Task exposing (Task)
import Html exposing (Html, h4, div, text, button, input)
import Html.Attributes exposing (id, type_)
import Html.Events exposing (onClick, targetValue, on)
import Time

import Http.Decorators exposing (..)
import Http exposing (..)


main =
    Html.program
        { init = init
        , update = update
        , view = view
        , subscriptions = \_ -> Sub.none
        }


type alias Model =
    { message : String
    }


init : (Model, Cmd Msg)
init = (Model "Initial state", Cmd.none)


type Msg
    = OneTask
    | SendManualReq
    | SendLessVerboseReq
    | TryUrlWithTime
    | TryUrlWithTime2
    | HandleResult (Result Error String)
    | HandleString String


-- Should resolve to something like "http://elm-lang.org?cacheBuster=12348257"
urlWithTime : Task x String
urlWithTime =
    cacheBusterUrl "http://elm-lang.org"


-- Should resolve to something like "http://elm-lang.org?param=7&cacheBuster=12348257"
urlWithTime2 : Task x String
urlWithTime2 =
    cacheBusterUrl "http://elm-lang.org?param=7"


oneTask : Task Error String
oneTask =
    taskWithCacheBuster (defaultGetString "http://elm-lang.org")


manualReq : RawRequest String
manualReq =
    { method = "GET"
    , headers = [header "X-Test-Header" "Foo"]
    , url = "http://apple.com"
    , body = Http.emptyBody
    , expect = Http.expectString
    , timeout = Nothing
    , withCredentials = False
    }

lessVerboseReq : RawRequest String
lessVerboseReq =
    let default = defaultGetString "http://debian.org"
    in {default | headers = [header "X-Test-Header" "Foo"]}


update : Msg -> Model -> (Model, Cmd Msg)
update msg model =
    case msg of
        OneTask ->
            ( { model | message = "Sent with sendWithCacheBuster" }
            , Task.attempt HandleResult oneTask
            )

        SendManualReq ->
            ( { model | message = "Sent manual req" }
            , sendRaw HandleResult manualReq
            )

        SendLessVerboseReq ->
            ( { model | message = "Sent less verbose req" }
            , sendRaw HandleResult lessVerboseReq
            )

        TryUrlWithTime ->
            ( model, Task.perform HandleString urlWithTime )

        TryUrlWithTime2 ->
            ( model, Task.perform HandleString urlWithTime2 )

        HandleResult result ->
            ( { model | message = toString result }
            , Cmd.none
            )

        HandleString result ->
            ( { model | message = result }
            , Cmd.none
            )


view : Model -> Html Msg
view model =
    div []
        [ button
            [ onClick OneTask ]
            [ text "sendWithCacheBuster" ]

        , button
            [ onClick SendManualReq ]
            [ text "Send Manual Req" ]

        , button
            [ onClick SendLessVerboseReq ]
            [ text "Send Less Verbose Req" ]

        , button
            [ onClick TryUrlWithTime ]
            [ text "Try urlWithTime" ]

        , button
            [ onClick TryUrlWithTime2 ]
            [ text "Try urlWithTime2" ]

        , h4 [] [ text "Message" ]
        , div [ id "message" ] [ text model.message ]
        ]

