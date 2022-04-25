# WebRTC Video Filter

Utility for [webrtc-cdn](https://github.com/AgustinSRG/webrtc-cdn) to apply a video filter with FFMpeg.

It uses [FFMpeg](https://ffmpeg.org/) for the video filtering, and the [pion/webrtc](https://github.com/pion/webrtc) for WebRTC connectivity.

## Compilation

In order to install dependencies, type:

```
go get github.com/AgustinSRG/webrtc-video-filter
```

To compile the code type:

```
go build
```

The build command will create a binary in the currenct directory, called `webrtc-video-filter`, or `webrtc-video-filter.exe` if you are using Windows.

## Usage

You can use the program from the command line:

```
webrtc-video-filter [OPTIONS] <SOURCE> <DESTINATION>
```

### SOURCE

The source must be a websocket URL of one of the webrtc-cdn nodes. Examples:

 - `ws://localhost/stream-id`
 - `wss://www.example.com/stream-id`

### DESTINATION

The destination must be a websocket URL of one of the webrtc-cdn nodes. Examples:

 - `ws://localhost/stream-id`
 - `wss://www.example.com/stream-id`

### OPTIONS

Here is a list of all the options:

| Option | Description |
|---|---|
| `--help, -h` | Shows the command line options |
| `--version, -v` | Shows the version |
| `--port, -p <port>` | Sets the port to use to forward the RTP packets to FFmpeg. By default, the port 400 is used. |
| `--video-filter, -vf <filter>` | Sets the video filter for FFmpeg |
| `--debug` | Enables debug mode (prints more messages) |
| `--ffmpeg-path <path>` | Sets the FFMpeg path. By default is `/usr/bin/ffmpeg`. You can also change it with the environment variable `FFMPEG_PATH` |
| `--auth-source, -as <auth-token>` | Sets auth token for the source. |
| `--auth-destination, -ad <auth-token>` | Sets auth token for the destination. |
| `--secret, -s <secret>` | Provides secret to generate authentication tokens. |

## WebRTC options

You can configure WebRTC configuration options with environment variables:

| Variable Name | Description |
|---|---|
| STUN_SERVER | STUN server URL. Example: `stun:stun.l.google.com:19302` |
| TURN_SERVER | TURN server URL. Set if the server is behind NAT. Example: `turn:turn.example.com:3478` |
| TURN_USERNAME | Username for the TURN server. |
| TURN_PASSWORD | Credential for the TURN server. |
