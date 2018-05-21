module Cast.Helpers exposing (evenElements, oddElements)

evenElements_ : List a -> Int -> List a
evenElements_ list index =
    case list of
        [] ->
            []

        hd :: tl ->
            if index % 2 == 0 then
                hd :: (evenElements_ tl (index + 1))
            else
                evenElements_ tl (index + 1)

evenElements : List a -> List a
evenElements list =
    evenElements_ list 1

oddElements_ : List a -> Int -> List a
oddElements_ list index =
    case list of
        [] ->
            []

        hd :: tl ->
            if index % 2 == 0 then
                oddElements_ tl (index + 1)
            else                
                hd :: (oddElements_ tl (index + 1))

oddElements : List a -> List a
oddElements list =
    oddElements_ list 1