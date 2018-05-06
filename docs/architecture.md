# Architecture

Maps:
  - excluded_sids
  - active_sid
  - available_sids
  - tty_writes

excluded_sids:
  - key: sid_t
  - value: int

active_sid:
  - key: int (always 0)
  - value: sid_t:
    - if sid_t doesn't exist, print nothing
    - if sid_t does exist and is 0, print everything
    - if sid_t does exist and equals current sid then print
    - if sid_t does exist and doesn't equal current sid, then exit

available_sids:
  - key: sid_t
  - value: uint64_t

Problems:
- If someone cats a binary, then ecents are missed
- Websockets

