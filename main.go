// Main

package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	child_process_manager "github.com/AgustinSRG/go-child-process-manager"
)

// Program entry point
func main() {
	// Read env vars
	ffmpegPath := os.Getenv("FFMPEG_PATH")

	if ffmpegPath == "" {
		ffmpegPath = "/usr/bin/ffmpeg"
	}

	// Read arguments
	args := os.Args

	if len(args) < 3 {
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			printHelp()
		} else if len(args) > 1 && (args[1] == "--version" || args[1] == "-v") {
			printVersion()
		} else {
			printHelp()
		}
		return
	}

	source := args[len(args)-2]

	uSource, err := url.Parse(source)
	if err != nil || (uSource.Scheme != "ws" && uSource.Scheme != "wss") {
		fmt.Println("The source is not a valid websocket URL")
		return
	}

	destination := args[len(args)-1]

	uDest, err := url.Parse(destination)
	if err != nil || (uDest.Scheme != "ws" && uDest.Scheme != "wss") {
		fmt.Println("The destination is not a valid websocket URL")
		return
	}

	protocolSource := uSource.Scheme
	hostSource := uSource.Host
	streamIdSource := ""

	if len(uSource.Path) > 0 {
		streamIdSource = uSource.Path[1:]
	} else {
		fmt.Println("The source URL must contain the stream ID. Example: ws://localhost/stream-id")
	}

	wsURLSource := url.URL{
		Scheme: protocolSource,
		Host:   hostSource,
		Path:   "/ws",
	}

	protocolDest := uDest.Scheme
	hostDest := uDest.Host
	streamIdDest := ""

	if len(uDest.Path) > 0 {
		streamIdDest = uDest.Path[1:]
	} else {
		fmt.Println("The destination URL must contain the stream ID. Example: ws://localhost/stream-id")
	}

	wsURLDest := url.URL{
		Scheme: protocolDest,
		Host:   hostDest,
		Path:   "/ws",
	}

	debug := false
	videoFilter := ""
	authTokenSource := ""
	authTokenDest := ""
	port := 4000

	for i := 1; i < (len(args) - 2); i++ {
		arg := args[i]

		if arg == "--debug" {
			debug = true
		} else if arg == "--ffmpeg-path" {
			if i == len(args)-3 {
				fmt.Println("The option '--ffmpeg-path' requires a value")
				return
			}
			ffmpegPath = args[i+1]
			i++
		} else if arg == "--video-filter" || arg == "-vf" {
			if i == len(args)-3 {
				fmt.Println("The option '--video-filter' requires a value")
				return
			}
			videoFilter = args[i+1]
			i++
		} else if arg == "--auth-source" || arg == "-as" {
			if i == len(args)-3 {
				fmt.Println("The option '--auth-source' requires a value")
				return
			}
			authTokenSource = args[i+1]
			i++
		} else if arg == "--auth-destination" || arg == "-ad" {
			if i == len(args)-3 {
				fmt.Println("The option '--auth-destination' requires a value")
				return
			}
			authTokenDest = args[i+1]
			i++
		} else if arg == "--port" || arg == "-p" {
			if i == len(args)-3 {
				fmt.Println("The option '--port' requires a value")
				return
			}
			port, err = strconv.Atoi(args[i+1])
			if err != nil || port <= 0 {
				fmt.Println("The option '--port' requires a numeric value")
				return
			}
			i++
		} else if arg == "--secret" || arg == "-s" {
			if i == len(args)-3 {
				fmt.Println("The option '--secret' requires a value")
				return
			}
			authTokenSource = generateToken(args[i+1], streamIdSource)
			authTokenDest = generateToken(args[i+1], streamIdDest)
			i++
		}
	}

	if _, err := os.Stat(ffmpegPath); err != nil {
		fmt.Println("Error: Could not find 'ffmpeg' at specified location: " + ffmpegPath)
		return
	}

	err = child_process_manager.InitalizeChildProcessManager()
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}
	defer child_process_manager.DisposeChildProcessManager()

	runProcess(wsURLSource, streamIdSource, wsURLDest, streamIdDest, ProcessOptions{
		debug:                debug,
		port:                 port,
		ffmpeg:               ffmpegPath,
		videoFilter:          videoFilter,
		authTokenSource:      authTokenSource,
		authTokenDestination: authTokenDest,
	})
}

func printHelp() {
	fmt.Println("Usage: webrtc-video-filter [OPTIONS] <SOURCE> <DESTINATION>")
	fmt.Println("    SOURCE: Can be a path to a video file or RTMP URL")
	fmt.Println("    DESTINATION: Websocket URL like ws(s)://host:port/stream-id")
	fmt.Println("    OPTIONS:")
	fmt.Println("        --help, -h                              Prints command line options.")
	fmt.Println("        --version, -v                           Prints version.")
	fmt.Println("        --port, -p <filter>                     Sets the port to use (By default 4000).")
	fmt.Println("        --video-filter, -vf <filter>            Sets video filter.")
	fmt.Println("        --debug                                 Enables debug mode.")
	fmt.Println("        --ffmpeg-path <path>                    Sets FFMpeg path.")
	fmt.Println("        --auth-source, -as <auth-token>         Sets authentication token for the source.")
	fmt.Println("        --auth-destination, -ad <auth-token>    Sets authentication token for the destination.")
	fmt.Println("        --secret, -s <secret>                   Sets secret to generate authentication tokens.")
}

func printVersion() {
	fmt.Println("webrtc-video-filter 1.0.0")
}

func killProcess() {
	os.Exit(0)
}
