# Logbee

Logbee is a process manager for Procfile-based applications, with live log-file output. A [neatnet](https://neatnet.tech) product, forked from [Hivemind](https://github.com/DarthSim/hivemind). At the moment, it supports Linux, FreeBSD, and macOS.

Procfile is a simple format to specify types of processes your application provides (such as web application server, background queue process, front-end builder) and commands to run those processes. It can significantly simplify process management for developers and is used by popular Platforms-as-a-Service, such as Heroku and Deis. You can learn more about the `Procfile` format [here](https://devcenter.heroku.com/articles/procfile).

There are some good Procfile-based process management tools, including [foreman](https://github.com/ddollar/foreman) by David Dollar, which started it all. The problem with most of those tools is that processes you want to manage start to think they are logging their output into a file, and that can lead to all sorts of problems: severe lagging, losing or breaking colored output. Tools can also add vanity information (unneeded timestamps in logs). [Hivemind](https://github.com/DarthSim/hivemind) was created to fix those problems once and for all. Logbee builds on it, adding the ability to persist the live log stream to a file.

## Enter Logbee

Logbee uses `pty` to capture process output. That fixes any problem with log clipping, delays, and TTY colors other process management tools may have.

## Installation

#### With Homebrew (macOS / Linux)

```bash
$ brew install neatnet/tap/logbee
```

Or tap first, then install:

```bash
$ brew tap neatnet/tap
$ brew install logbee
```

#### Download a release binary

Prebuilt binaries for Linux, macOS, and FreeBSD are attached to each
[release](https://github.com/neatnet/logbee/releases/latest).

#### From Source

You need Go 1.18 or later to build the project.

```bash
$ go install github.com/neatnet/logbee@latest
```

## Usage

Logbee works with a `Procfile`. It may look like this:

```Procfile
web: bin/rails server
worker: bundle exec sidekiq
assets: gulp watch
```

To get started, you just need to run Logbee from your working directory containing `Procfile`.

```bash
$ logbee
```

If `Procfile` isn't located in your working directory, or named it non-standard as `Procfile.dev`, you can specify the path to it: [**Fun Fact:** Name of the `Procfile` is arbitrary and can be anything, although it is a best practice to name it as `Procfile` for sanity]

```bash
$ logbee path/to/your/Procfile
$ logbee path/to/your/Procfile.dev
```

Run `logbee --help` to see other options. Note that every Logbee option can be set with corresponding environment variable.

### Writing logs to a file

By default Logbee only prints to the console. Pass `--log-file` (`-L`, env `LOGBEE_LOG_FILE`) to also write the aggregated output stream to a file, live and line-by-line, so you can `tail -f` it or feed it to an external analytics/LLM process:

```bash
$ logbee --log-file dev.log
$ tail -f dev.log
web    | listening on :3000
worker | booted
```

The file is plain, greppable text: the `name | ` prefix carries no color escapes and any ANSI codes in the process output are stripped (the console stays fully colored). Parent directories are created automatically. With `--print-timestamps` each line is prefixed `15:04:05 web | ...`.

The file is truncated on start by default; pass `--log-append` (env `LOGBEE_LOG_APPEND`) to append to an existing file instead.

### Environment

If you need to set specific environment variables before running a `Procfile`, you can specify them in the `.env` file in the current working directory. The file should contain `variable=value` pairs, one per line:

```
PATH=$PATH:/additional/path
PORT=3000
LOGBEE_TITLE=my_awsome_app
```

## Author

Logbee is a [neatnet](https://neatnet.tech) product, forked from [Hivemind](https://github.com/DarthSim/hivemind) by Sergey "DarthSim" Aleksandrovich.

Highly inspired by [Foreman](https://github.com/ddollar/foreman).

## License

Logbee is licensed under the MIT license.

See LICENSE for the full license text.
