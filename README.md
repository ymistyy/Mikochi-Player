# Mikochi Player

A small web-based video player made to work alongside [Mikochi](https://github.com/zer0tonin/Mikochi).

I really liked Mikochi as a simple self-hosted file browser,  I wanted a other way to watch videos from it directly in a browser, especially on phones and other devices.
So I made this!

> **This is the official Mikochi project.**
>
> I am not affiliated with, connected to, or endorsed by the Mikochi project or its author.
> I simply liked the project and wanted something like this for my own setup. I figured other people might find it useful too!

---

## Screenshots

![Video player](https://imgur.com/4qhhTRh.png)

---

## What it does

- Browse files and folders from your Mikochi server
- Open videos directly in your browser
- Video seeking with HTTP range requests
- Automatic subtitle detection
- Supports `.srt` and `.vtt` subtitles
- Automatically looks for subtitles in the same folder as the video
- Multiple subtitle tracks can be selected
- SRT files are converted to WebVTT for browser playback
- Supports a wide range of video file extensions

---

## Example

If your folder contains:

    Movies/
    ├── Example Movie.mkv
    ├── Example Movie.srt
    └── Example Movie.en.srt

The player will detect the subtitle files automatically and make them available in the video player's **Subtitles** menu.

---

# Requirements

- Go 1.22 or newer
- A running Mikochi server
- A Mikochi username and password

You can find the original Mikochi project here:

**https://github.com/zer0tonin/Mikochi**

---

# Installation

Clone the repository:

    git clone https://github.com/ymistyy/Mikochi-Player
    cd mikochi-player

Edit it:

    nano config.json

Example:

    {
      "listen": "0.0.0.0:8090",
      "mikochi_url": "Whatever the URL or IP is",
      "mikochi_username": "YOUR_MIKOCHI_USERNAME",
      "mikochi_password": "YOUR_MIKOCHI_PASSWORD",
      "session_ttl_hours": 24
    }

Run it:

    go run .

Or build a binary:

    go build -o mikochi-player .

Then run:

    ./mikochi-player

---

# Running it as a system service

If you want it to run permanently as a web server, you can use systemd.

Build the application:

    go build -o /opt/mikochi-player/mikochi-player .

Make sure the directory contains:

    /opt/mikochi-player/
    ├── mikochi-player
    ├── config.json
    └── static/

Create a systemd service:

    sudo nano /etc/systemd/system/mikochi-player.service

Use:

    [Unit]
    Description=Mikochi Player
    After=network.target

    [Service]
    Type=simple
    WorkingDirectory=/opt/mikochi-player
    ExecStart=/opt/mikochi-player/mikochi-player
    Restart=on-failure
    RestartSec=5

    [Install]
    WantedBy=multi-user.target

Then:

    sudo systemctl daemon-reload
    sudo systemctl enable --now mikochi-player

Check the status:

    sudo systemctl status mikochi-player

View the logs:

    journalctl -u mikochi-player -f

The player should now start automatically when the machine boots.

---

# Configuration

`config.json` contains the connection information for Mikochi.

    {
      "listen": "0.0.0.0:8090",
      "mikochi_url": "Whatever the URL or IP is",
      "mikochi_username": "YOUR_MIKOCHI_USERNAME",
      "mikochi_password": "YOUR_MIKOCHI_PASSWORD",
      "session_ttl_hours": 24
    }

### `listen`

The address and port the player listens on.

For LAN access:

    "0.0.0.0:8090"

### `mikochi_url`

The URL of your Mikochi installation.

Example:

    "http://192.168.1.21" or "http://mikochi.home"

### `mikochi_username`

Your Mikochi username.

### `mikochi_password`

Your Mikochi password.

### `session_ttl_hours`

How long a browser session remains logged in.

---

# Security

The player is intended primarily for private networks and self-hosted environments.

Mikochi credentials are stored in `config.json` and are never sent to the browser.

The browser receives its own session cookie while the Mikochi authentication token stays on the server.

Do **not** expose `config.json` publicly.

---

# Video formats

The player recognizes a number of video formats, including:

- MP4
- M4V
- WebM
- OGG / OGV
- MOV
- MKV
- AVI
- WMV
- FLV
- MPEG
- MPG
- TS
- MTS
- M2TS
- 3GP
- 3G2
- VOB

Whether a particular file actually plays depends on the codecs supported by the browser.

For example, a browser may recognize an `.mkv` file but still be unable to decode the video or audio codec inside it.

---

# Subtitles

Subtitles are automatically detected when they are stored next to the video.

For example:

    Movie.mkv
    Movie.srt

or:

    Movie.mkv
    Movie.en.srt
    Movie.fr.srt

The player will show the available subtitle tracks in the **Subtitles** menu.

SRT subtitles are converted to WebVTT on the server because WebVTT is supported directly by most browsers.

---

# Why I made this

Mikochi already does a great job at being a lightweight file browser and it can generate streaming links for VLC.

I mainly wanted something a little more convenient for watching things from a phone, tablet or another computer without having to copy a streaming URL to VLC because i mainly use it for movies.

This project is just my fix to that.

---

# Credits

**Original project:**

https://github.com/zer0tonin/Mikochi

Mikochi is created and maintained by its original authors. Please visit the original repository for the actual Mikochi project, documentation, source code and license information.

This project is independentand is not affiliated with or endorsed by the Mikochi project.

---

# License / Disclaimer

This project is provided free of charge and without warranty.

You are responsible for securing your own server, network, Mikochi installation and credentials.

This project does not replace or modify the original Mikochi software.

Please check the original Mikochi repository and its license for information about Mikochi itself:

https://github.com/zer0tonin/Mikochi

---

## Final note

If you find a bug or have an idea, feel free to open an issue or contribute.
