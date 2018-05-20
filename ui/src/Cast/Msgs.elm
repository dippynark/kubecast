module Cast.Msgs exposing (..)

import Navigation exposing (Location)

type Msg
    = DisplayCast String
    | ListCasts String
    | OnLocationChange Location