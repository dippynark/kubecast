module Cast.View exposing (..)

import Html exposing (..)
import Html.Events exposing (..)
import Html.Attributes exposing (..)
import String.Extra exposing (..)

import Cast.Msgs exposing (Msg)
import Cast.Models exposing (Model)

castOption : a -> b -> Html Msg
castOption cast labels =
    option [ value (toString cast) ] [ text (unquote (toString labels)) ]

view : Model -> Html Msg
view model =
    div [ class "container" ] [
        div [ class "row" ] [
            h2 [ class "text-center" ] [ text "Terminal Sessions" ]
            , select [ onInput Cast.Msgs.DisplayCast ]
                (List.map2 castOption model.casts model.labels)
        ], div [ class "row" ] [
            div [ id "asciinema-player-container", class "container" ] [ ]
        ]
    ]