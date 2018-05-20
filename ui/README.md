# Elm

## Initialise

- elm package install --yes elm-lang/core
- elm package install --yes elm-lang/html

## Running

- cd ~/go/src/github.com/auth0-blog/nodejs-jwt-authentication-sample; node server.js

## Types:

```
divide x y = x / y
divide x = \y -> x / y
divide = \x -> (\y -> x / y)

divide 3 2 
(divide 3) 2
((\x -> (\y -> x / y)) 3) 2
(\y -> 3 / y) 2
3/2
1.5
```
