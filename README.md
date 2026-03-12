# goremotetype

A self-contained Go binary that serves a web page with a textarea on your local
network, then relays keyboard and composition events from that textarea to your
Linux X11 desktop via the XTEST extension. Whatever you type into the page
appears wherever your cursor is on your computer.

This is my second attempt at a dictation tool (the first being
[talkxtyper](https://github.com/leafo/talkxtyper)). I realized that I liked
Gboard's built in dictation, it's accurate and responsive. I use my phone as
the microphone and this tool to forward the dictation "compose" events
generated in the browser to the local computer so you can speak prose, and see
the edits/typing it's doing in real time on your computer.

This tool currently doesn't try to handle any of the proper noun conversation
that talkxtyper tried to address.

## Requirements

- Linux with X11
- `libX11` and `libXtst` development headers (for building)

## Build & run

```
go build -o goremotetype .
./goremotetype
```

On startup it prints the LAN URL:

```
goremotetype listening on http://localhost:8088
goremotetype LAN URL: http://192.168.1.100:8088
```

### Options

```
-listen  listen address (default "0.0.0.0:8088")
-key-delay-ms  delay in milliseconds between injected X11 key presses
-tray    show a system tray icon; disable with -tray=false
```

For apps that drop or corrupt very fast synthetic input, try a small server-side
delay such as:

```
./goremotetype -key-delay-ms=2
```
