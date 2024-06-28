# swo command-line client for SolarWinds Observability platform

Small standalone command line tool to retrieve and search recent app
server logs from [Solarwinds].

### This is v1 of the swo-cli and it supports ONLY logs search.
### This is v1 of the swo-cli and it DOES NOT support tailing.

Supports optional Boolean search queries. Example:

    $ swo logs "(www OR db) (nginx OR pgsql) -accepted"

## Quick Start

Install [Go]

    $ go install github.com/solarwinds/swo-cli/cmd/swo@latest
    $ echo "token: 123456789012345678901234567890ab" > ~/.swo-cli.yml
    $ echo "api-url: https://api.na-01.cloud.solarwinds.com" >> ~/.swo-cli.yml
    $ swo

Retrieve the full-access token from SolarWinds Observability.

The API token can also be passed in the `SWO_API_TOKEN`
environment variable instead of a configuration file. Example:

    $ export SWO_API_TOKEN='123456789012345678901234567890ab'
    $ swo logs


## Configuration

Create ~/.swo-cli.yml containing your full-access API token and API URL, or specify the
path to that file with -c. Example (from
examples/swo-cli.yml.example):

    token: 123456789012345678901234567890ab
    api-url: https://api.na-01.cloud.solarwinds.com

Retrieve token from SolarWinds Observability page (`Settings` -> `API Tokens` -> `Create API Token` -> `Full Access`).

## Usage & Examples

```
$ swo --help

NAME:
swo - SolarWinds Observability Command-Line Interface

USAGE:
swo [global options] command [command options]

VERSION:
v1.0.0

COMMANDS:
logs     command-line search for SolarWinds Observability log management service
help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
--api-url value           URL of the SWO API (default: "https://api.na-01.cloud.solarwinds.com")
--api-token value         API token
--config value, -c value  path to config (default: "~/.swo-cli.yml")
--help, -h                show help
--version, -v             print the version
```

### Count, pivot, and summarize

To count the number of matches, pipe to `wc -l`. For example, count how
many logs contained `Failure` in the last minute:

    $ swo logs --min-time '1 minute ago' Failure | wc -l
    42

Output only the program/file name (which is output as field 5):

    $ swo logs --min-time '1 minute ago' | cut -f 5 -d ' '
    passenger.log:
    sshd:
    app/web.2:

Count by source/system name (field 4):

    $ swo logs --min-time '1 minute ago' | cut -f 4 -d ' ' | sort | uniq -c
      98 www42
      39 acmedb-core01
      2 fastly

For sum, mean, and statistics, see
[datamash](http://www.gnu.org/software/datamash/) and [one-liners](https://www.gnu.org/software/datamash/alternatives/).

### Colors

ANSI color codes are retained, so log messages which are already colorized
will automatically render in color on ANSI-capable terminals.

For content-based colorization, pipe through [lnav]. Install `lnav` from your
preferred package repository, such as `brew install lnav` or
`apt-get install lnav`, then:

    $ swo logs | lnav
    $ swo logs --min-time "1 hour ago" error | lnav

### Redirecting output

Since output is line-buffered, pipes and output redirection will automatically
work:

    $ swo logs | less
    $ swo logs --min-time '2016-01-15 10:00:00' > logs.txt

If you frequently pipe output to a certain command, create a function which
accepts optional arguments, invokes `swo` with any arguments, and pipes
output to that command. For example, this `swocolor` function will pipe to `lnav`:

    $ function swocolor() { swo logs $* | lnav; }

Add the `function` line to your `~/.bashrc`. It can be invoked with search
parameters:

    $ swocolor 1.2.3 Failure

### Negation-only queries

Unix shells handle arguments beginning with hyphens (`-`) differently
([why](http://unix.stackexchange.com/questions/11376/what-does-double-dash-mean)).
Usually this is moot because most searches start with a positive match.
To search only for log messages without a given string, use `--`. For
example, to search for `-whatever`, run:

    swo logs -- -whatever

### Time zones

Times are interpreted in the client itself, which means it uses the time
zone that your local PC is set to. Log timestamps are also output in the
same local PC time zone.

When providing absolute times, append `UTC` to provide the input time in
UTC. For example, regardless of the local PC time zone, this will show
messages beginning from 1 PM UTC:

    swo logs --min-time "2024-04-27 13:00:00 UTC"

Output timestamps will still be in the local PC time zone.

### Quoted phrases

Because the Unix shell parses and strips one set of quotes around a
phrase, to search for a phrase, wrap the string in both single-quotes
and double-quotes. For example:

    swo logs '"Connection reset by peer"'

Use one set of double-quotes and one set of single-quotes. The order
does not matter as long as the pairs are consistent.

Note that many phrases are unique enough that searching for the
words yields the same results as searching for the quoted phrase. As a
result, quoting strings twice is often not actually necessary. For
example, these two searches are likely to yield the same log messages,
even though one is for 4 words (AND) while the other is for a phrase:

    swo logs Connection reset by peer
    swo logs '"Connection reset by peer"'

### Multiple API tokens

To use multiple API tokens (such as for separate home and work SolarWinds Observability 
accounts), create a `.swo-cli.yml` configuration file in each project's
working directory and invoke the CLI in that directory. The CLI checks for
`.swo-cli.yml` in the current working directory prior to using
`~/.swo-cli.yml`.

Alternatively, use shell aliases with different `-c` paths. For example:

    echo "alias swo1='swo logs -c /path/to/swo-cli-home.yml'" >> ~/.bashrc
    echo "alias swo2='swo logs -c /path/to/swo-cli-work.yml'" >> ~/.bashrc


### Build

1. Bump `Version` in `version/version.go`
2. Build the swo CLI: `$ go build ./cmd/swo`

### Install & Test

1. Download repository: `$ git clone https://github.com/solarwinds/swo-cli.git`
2. Build the binary: `$ go build ./cmd/swo`
3. Test: `$ ./swo logs test search string`

### Release

1. Bump `version` in `cmd/swo/main.go`
2. Bump tag on main branch
3. Push to upstream

## Contribute

Testing:

Run all the tests with `go test -v -count=1 ./...`
Run go linter with `make ci-lint`

Bug report:

1. See whether the issue has already been reported:
   http://github.com/solarwinds/swo-cli/issues/
2. If you don't find one, create an issue with a repro case.

Enhancement or fix:

1. Fork the project:
   http://github.com/solarwinds/swo-cli
2. Make your changes with tests.
3. Commit the changes without changing the version/version.go file.
4. Send a pull request.

[Solarwinds]: https://my.na-01.cloud.solarwinds.com/
[lnav]: http://lnav.org/
[escape characters]: http://en.wikipedia.org/wiki/ANSI_escape_code#Colors
[Go]: https://go.dev/doc/install
