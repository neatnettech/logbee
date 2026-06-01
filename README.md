# Logbee

Logbee is a process manager for Procfile-based applications, with live log-file output. A [neatnettech](https://neatnet.tech) product, forked from [Hivemind](https://github.com/DarthSim/hivemind). At the moment, it supports Linux, FreeBSD, and macOS.

Procfile is a simple format to specify types of processes your application provides (such as web application server, background queue process, front-end builder) and commands to run those processes. It can significantly simplify process management for developers and is used by popular Platforms-as-a-Service, such as Heroku and Deis. You can learn more about the `Procfile` format [here](https://devcenter.heroku.com/articles/procfile).

There are some good Procfile-based process management tools, including [foreman](https://github.com/ddollar/foreman) by David Dollar, which started it all. The problem with most of those tools is that processes you want to manage start to think they are logging their output into a file, and that can lead to all sorts of problems: severe lagging, losing or breaking colored output. Tools can also add vanity information (unneeded timestamps in logs). [Hivemind](https://github.com/DarthSim/hivemind) was created to fix those problems once and for all. Logbee builds on it, adding the ability to persist the live log stream to a file.

## Enter Logbee

Logbee uses `pty` to capture process output. That fixes any problem with log clipping, delays, and TTY colors other process management tools may have.

## Installation

#### With Homebrew (macOS / Linux)

```bash
$ brew install neatnettech/tap/logbee
```

Or tap first, then install:

```bash
$ brew tap neatnettech/tap
$ brew install logbee
```

#### Download a release binary

Prebuilt binaries for Linux, macOS, and FreeBSD are attached to each
[release](https://github.com/neatnettech/logbee/releases/latest).

#### From Source

You need Go 1.18 or later to build the project.

```bash
$ go install github.com/neatnettech/logbee@latest
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

### Driving an interactive process (Expo, Metro, etc.)

By default Logbee does not forward your keyboard to managed processes, so interactive dev servers that listen for keypresses (Expo/Metro: `r` to reload, `i`/`a` to open iOS/Android, `j` for the debugger) can't be controlled. Pass `--interactive` (`-i`, env `LOGBEE_INTERACTIVE`) with the process name to forward your terminal's stdin to that one process:

```bash
$ logbee --interactive expo
```

While an interactive process runs, Logbee puts your terminal into raw mode so single keypresses reach the process immediately (no need to press Enter). `Ctrl-C` is forwarded to the interactive process; when it exits, Logbee shuts the rest down as usual. Only one process can be interactive at a time, and its name must match an entry that's actually launched (respecting `--processes`).

### Interactive console (`--tui`)

Pass `--tui` (`-u`, env `LOGBEE_TUI`) for a full-screen console instead of a flat stream. You get **one tab per process plus an aggregate `all` tab** (the same color-prefixed, interleaved view as the default output):

```bash
$ logbee --tui
$ logbee --tui -i metro path/to/Procfile   # start with the metro tab live
```

The console requires a terminal. When stdout is piped or redirected, Logbee prints a warning and falls back to plain streaming, so pipes, `--log-file`, and CI stay unaffected. Use `--scrollback` (env `LOGBEE_SCROLLBACK`, default `5000`) to set how many log lines each tab keeps.

**Navigation**

| Key | Action |
|-----|--------|
| `←` / `→`, `Tab` / `Shift+Tab` | Switch tabs |
| `0`–`9` | Jump to a tab by index (`0` = aggregate) |
| `↑` / `↓`, `PgUp` / `PgDn`, `Home` / `End` | Scroll the active tab |
| `Home` / `End`, `g` / `G` | Jump to top / bottom |
| `f` | Toggle follow (auto-scroll to newest) |
| `i` / `Enter` | Go **live** on the current process tab |
| `?` | Toggle the full key-map overlay |
| `q`, `Ctrl-C` | Quit |

The tab bar sits along the **bottom** of the console (log output fills the space above it). Tabs show `●` when live and `*` once their process has exited. Press `?` at any time for the full key map.

**Live mode — driving a dev server**

On a process tab, press `i` (or start it live with `-i <name>`) to forward every keystroke straight to that process — drive Expo/Metro, Vite, or any interactive CLI exactly as if you ran it directly (`r` reload, `i`/`a` iOS/Android, `j` debugger, …).

While live, the leader key **`Ctrl-B`** (tmux-style — Expo/Metro don't use it) gives you Logbee commands without leaving the process:

| Sequence | Action |
|----------|--------|
| `Ctrl-B` then `0`–`9` | Jump to a tab |
| `Ctrl-B` then `q` | Quit Logbee |
| `Ctrl-B` then `i` | Stop live (back to scroll/nav) |
| `Ctrl-B` then `?` | Show the key-map overlay |
| `Ctrl-B` then `Ctrl-B` | Send a literal `Ctrl-B` to the process |

`Ctrl-C` while live is forwarded to the process (so the dev server handles it); quit Logbee itself with `Ctrl-B` `q`.

### Environment

If you need to set specific environment variables before running a `Procfile`, you can specify them in the `.env` file in the current working directory. The file should contain `variable=value` pairs, one per line:

```
PATH=$PATH:/additional/path
PORT=3000
LOGBEE_TITLE=my_awsome_app
```

## Author

Logbee is a [neatnettech](https://neatnet.tech) product, forked from [Hivemind](https://github.com/DarthSim/hivemind) by Sergey "DarthSim" Aleksandrovich.

Highly inspired by [Foreman](https://github.com/ddollar/foreman).

## License

Logbee is licensed under the MIT license.

See LICENSE for the full license text.
