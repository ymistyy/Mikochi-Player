# Mikochi Player

A small web-based video player made to work alongside [Mikochi](https://github.com/zer0tonin/Mikochi).

I really liked Mikochi as a simple self-hosted file browser,  I wanted a other way to watch videos from it directly in a browser, especially on phones and other devices.
So I made this!

It is basically a small companion app that sits with Mikochi and provides a simple video player while keeping Mikochi as the source of the files.

> **This is not an official Mikochi project.**
>
> I am not affiliated with, connected to, or endorsed by the Mikochi project or its author.
> I simply liked the project and wanted something like this for my own setup. I figured other people might find it useful too!

---

## Screenshots

![Video player](https://i.imgur.com/yW9LMiM.png)

---

## What it does

- Browse files and folders from your Mikochi server
- Open videos directly in your browser
- Works on desktop and mobile
- Video seeking with HTTP range requests
- Automatic subtitle detection
- Supports `.srt` and `.vtt` subtitles
- Automatically looks for subtitles in the same folder as the video
- Multiple subtitle tracks can be selected
- SRT files are converted to WebVTT for browser playback
- Supports a wide range of video file extensions
- Keeps the Mikochi credentials on the server

---

## Example

If your folder contains:

    Movies/
    ├── Example Movie.mkv
    ├── Example Movie.srt
    ├── Example Movie.nl.srt
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

## Windows

Download or clone this repository.

Open PowerShell in the project directory:

    cd C:\path\to\mikochi-player

Edit `config.json`:

    {
      "listen": "0.0.0.0:8090",
      "mikochi_url": "Whatever the URL or IP is",
      "mikochi_username": "YOUR_MIKOCHI_USERNAME",
      "mikochi_password": "YOUR_MIKOCHI_PASSWORD",
      "session_ttl_hours": 24
    }

Then start the player:

    go run .

You should see something similar to:

    Mikochi Player listening on http://0.0.0.0:8090

Open:

    http://127.0.0.1:8090

If you want to access it from another device on your network, use the IP address of the computer running the player:

    http://192.168.1.100:8090

You may need to allow the port through Windows Firewall.

---

# Linux

The player is just a normal Go application, so it can run on basically any Linux machine where Go is available.

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

# Reverse proxy

The player can also be placed behind a reverse proxy such as nginx, Caddy or Traefik.

For example:

    Browser
        |
        v
    HTTPS / reverse proxy
        |
        v
    Mikochi Player :8090
        |
        v
    Mikochi

This is useful if you want to access the player through a proper hostname instead of a port.

For example:

    https://player.example.com

The application itself does not require a reverse proxy and works perfectly fine directly on a LAN.

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

For local-only access:

    "127.0.0.1:8090"

### `mikochi_url`

The URL of your Mikochi installation.

Example:

    "http://192.168.1.21" or "http://mikochi.home.arpa"

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

The player recognizes a number of common video formats, including:

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
    Movie.nl.srt
    Movie.en.srt
    Movie.fr.srt

The player will show the available subtitle tracks in the **Subtitles** menu.

SRT subtitles are converted to WebVTT on the server because WebVTT is supported directly by modern browsers.

---

# Why I made this

Mikochi already does a great job at being a lightweight file browser and it can generate streaming links for VLC/MPV.

I mainly wanted something a little more convenient for watching things from a phone, tablet or another computer without having to copy a streaming URL to VLC because i mainly use it for movies.

This project is just my fix to that.

Thank you Mikochi!

---

# Credits

This project would not exist without Mikochi.

**Original project:**

https://github.com/zer0tonin/Mikochi

Mikochi is created and maintained by its original authors. Please visit the original repository for the actual Mikochi project, documentation, source code and license information.

This project is independentand is not affiliated with or endorsed by the Mikochi project.

---

# License / Disclaimer

This project is provided free of charge and without warranty.

Use it at your own risk.

I am not responsible for:

- Data loss
- Damaged or corrupted files
- Server downtime
- Security issues caused by incorrect configuration
- Exposing your Mikochi installation to the internet
- Problems caused by third-party software
- Video playback issues caused by unsupported codecs
- Any damage resulting from the use or misuse of this software

You are responsible for securing your own server, network, Mikochi installation and credentials.

This project does not replace or modify the original Mikochi software.

Please check the original Mikochi repository and its license for information about Mikochi itself:

https://github.com/zer0tonin/Mikochi

---

## Final note

This is a small personal project that I decided to share because I thought someone else using Mikochi might find it useful too.

If you find a bug or have an idea, feel free to open an issue or contribute.
