module Cast.View exposing (..)

import Html exposing (..)
import Html.Events exposing (..)
import Html.Attributes exposing (..)
import String.Extra exposing (..)

import Cast.Msgs exposing (Msg)
import Cast.Models exposing (Model)

castOption : a -> Html Msg
castOption cast =
    option [ value (toString cast) ] [ text (unquote (toString cast)) ]

view : Model -> Html Msg
view model =
    div [ class "container" ] [
        div [ class "row" ] [
            h2 [ class "text-center" ] [ text "Terminal Sessions" ]
            , select [ onInput Cast.Msgs.DisplayCast ]
                (List.map castOption model.casts)
        ], div [ class "row" ] [
            div [ id "asciinema-player-container", class "container" ] [ ]
        ]
    ]