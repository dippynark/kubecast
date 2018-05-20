module Cast.Models exposing (..)

import Navigation exposing (Location)

type Route
    = HomeRoute
    | NotFoundRoute

type alias Model =
    { cast : String
    , casts : List String
    , location : Location
    }

initialModel : Location -> Model
initialModel location =
    { cast = ""
    , casts = []
    , location = location
    }