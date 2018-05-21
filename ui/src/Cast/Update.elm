port module Cast.Update exposing (..)

import Cast.Msgs exposing (Msg)
import Cast.Models exposing (Model)
import Array exposing (fromList, get)
import String exposing (trim)
import Cast.Helpers exposing (oddElements, evenElements)

update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Cast.Msgs.DisplayCast cast ->
            ( { model | cast = cast }, displayCast cast)
        Cast.Msgs.ListCasts castsString ->
            let
                casts = oddElements (String.split "\n" (trim castsString))
                labels = evenElements (String.split "\n" (trim castsString))
                (cast, command) = setCastIfEmpty model.cast casts 
            in
                ( { model | cast = cast, casts = casts, labels = labels}, command)
        Cast.Msgs.OnLocationChange location ->
            ( { model | location = location }, Cmd.none)

port displayCast : String -> Cmd msg

setCastIfEmpty : String -> List String -> (String, Cmd Msg)
setCastIfEmpty currentCast newCasts =
    if currentCast == "" then
        case get 0 (fromList newCasts) of
            Just cast ->
                (cast, displayCast cast)
            Nothing ->
                ("", Cmd.none)
    else
        (currentCast, Cmd.none)