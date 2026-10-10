# Compatibility suite

A black-box test of GOP Server's observable behaviour. It starts a server binary,
sends it real HTTP requests and compares the responses with golden files recorded
from `main` before the rewrite. Any server that should behave the same can be
pointed at it.

## Running

```sh
make compat                  # build bin/gop-server, run the suite
make compat-update           # re-record the golden files
```

or directly:

```sh
cd test/compat
GOP_SERVER_BIN=/path/to/gop-server go test -count=1 ./...
go test ./... -run 'TestCompat/default/sessions' -args -update
```

| Variable | Default | Meaning |
|---|---|---|
| `GOP_SERVER_BIN` | builds `./src` from the repository root | server binary under test |
| `GOP_COMPAT_OUT` | unset | on a mismatch, write the actual output here (CI uploads it) |

## How it works

Each scenario in `cases_test.go` copies the site into a temp directory, writes
`config/configuration.json` with a free port plus the scenario's overrides, and
starts the server there. Cases run in order against that one server, so the
cache carries over: a script's first request records `X-Cache-Status: MISS`, the
repeat records `HIT`.

| Scenario | Config | Covers |
|---|---|---|
| `default` | as shipped | every example script, directory index and listing, GET/POST/multipart parameters, cookies, sessions, request headers, basic auth, methods, the `/api/` alias, uploads, static files, 404s, `/gop-info/`, `/server-status/` |
| `denied` | `DenyFrom: 127.0.0.0/8` | the IP-denied script |
| `ratelimit` | 2 requests per day | the rate-limited script |

`testdata/fixtures` is copied into the document root as `/__compat/` and covers
helpers no example uses (`_status_code`, `_setcookie`, `_session_destroy`,
`_base64_*`, `_json_encode`, `substr`, …) and the `_HEADER` map.

A golden file lists each request (`===`, with request headers as `>`) and its
response: status (`---`), sorted headers, body. `TestCasesCoverExamples` fails if
a script is added to `public/examples` without a case for it.

## Normalisation

Kept to what genuinely changes between runs:

- `Date`, `Content-Length` and `Transfer-Encoding` headers are dropped; `Last-Modified` becomes `{{HTTP-DATE}}`.
- The temp site path becomes `{{ROOT}}`, `127.0.0.1:<port>` becomes `{{HOST}}`, `REMOTE_PORT` and the gop-info port become placeholders.
- 64-character hex strings (`FILE_UPLOAD_KEY`) become `{{SHA256}}`; `temp-N.go` and `go-buildN` names are masked.
- Per case: random numbers in `go_file_with_functions` and `for_loop_100000`, the user name from `whoami`, upload file names, and the clock and counters on `/server-status/`.
- Non-UTF-8 bodies are recorded as length and SHA-256.

Compiler error pages are recorded verbatim, so goldens are tied to the Go minor
version CI pins (1.24).

## Not replayed

`errors/gop_file_while_true.gop` and `errors/go_file_while_true.go` are listed but
skipped: on a cache miss the build step has no timeout, so the request hangs until
`WriteTimeout` and the `for {}` process keeps running and buffering output. Once
the rewrite adds build timeouts they should get real cases.

## Changing behaviour on purpose

The goldens record today's behaviour, warts included (for example `_GET` also
holds POST values, multipart text fields are missing from `_POST`, any client can
set `_SESSION`). A PR that changes one of these deliberately re-records the
affected files with `-update` and the diff shows exactly what changed.
