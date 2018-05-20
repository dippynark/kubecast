module Main exposing (..)

import WebSocket
import Navigation exposing (Location)
import Cast.Msgs exposing (Msg)
import Cast.Models exposing (Model, initialModel)
import Cast.Update exposing (update)
import Cast.View exposing (view)
    
init : Location -> (Model, Cmd Msg)
init location =
    ( initialModel location, Cmd.none )

webSocketUrl : Location -> String -> String
webSocketUrl location path =
    case ( location.protocol, location.host ) of
        -- Local development proxy doesn't support Websockets
        ( _, "localhost:3000" ) ->
            "ws://localhost:5050" ++ path

        ( _, "127.0.0.1:3000" ) ->
            "ws://127.0.0.1:5050" ++ path

        -- TLS should use wss 
        ( "https:", _ ) ->
            "wss://" ++ location.host ++ path

        -- Non-TLS should use ws
        _ ->
            "ws://" ++ location.host ++ path

subscriptions : Model -> Sub Msg
subscriptions model =
  WebSocket.listen (webSocketUrl model.location "/list") Cast.Msgs.ListCasts

main : Program Never Model Msg
main =
    Navigation.program Cast.Msgs.OnLocationChange
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }