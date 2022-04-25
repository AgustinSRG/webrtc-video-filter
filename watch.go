// Code to receive the remote video track

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type ProcessOptions struct {
	port                 int
	debug                bool
	ffmpeg               string
	videoFilter          string
	authTokenSource      string
	authTokenDestination string
}

func runProcess(source url.URL, sourceStreamId string, destination url.URL, destinationStreamId string, options ProcessOptions) {
	// Mutex
	lock := sync.Mutex{}

	m := &webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	// Create a InterceptorRegistry. This is the user configurable RTP/RTCP Pipeline.
	// This provides NACKs, RTCP Reports and other features. If you use `webrtc.NewPeerConnection`
	// this is enabled by default. If you are manually managing You MUST create a InterceptorRegistry
	// for each PeerConnection.
	i := &interceptor.Registry{}

	// Use the default set of Interceptors
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))

	// Connect to websocket
	if options.debug {
		fmt.Println("Connecting to " + source.String())
	}
	c, _, err := websocket.DefaultDialer.Dial(source.String(), nil)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer c.Close()

	go func() {
		for {
			time.Sleep(20 * time.Second)

			// Send hearbeat message
			heartbeatMessage := SignalingMessage{
				method: "HEARTBEAT",
				params: nil,
				body:   "",
			}

			lock.Lock()
			sendErr := c.WriteMessage(websocket.TextMessage, []byte(heartbeatMessage.serialize()))
			lock.Unlock()

			if options.debug {
				fmt.Println("[SOURCE] >>>\n" + string(heartbeatMessage.serialize()))
			}

			if sendErr != nil {
				return
			}
		}
	}()

	// Send play message
	pubMsg := SignalingMessage{
		method: "PLAY",
		params: make(map[string]string),
		body:   "",
	}
	pubMsg.params["Request-ID"] = "play01"
	pubMsg.params["Stream-ID"] = sourceStreamId
	if options.authTokenSource != "" {
		pubMsg.params["Auth"] = options.authTokenSource
	}
	c.WriteMessage(websocket.TextMessage, []byte(pubMsg.serialize()))

	if options.debug {
		fmt.Println("[SOURCE] >>>\n" + string(pubMsg.serialize()))
	}

	receivedOffer := false
	receivedVideoTrack := false
	closed := false

	var peerConnection *webrtc.PeerConnection = nil

	// Read websocket messages
	for {
		if closed {
			return
		}
		func() {
			_, message, err := c.ReadMessage()
			if err != nil {
				closed = true
				killProcess()
				return // Closed
			}

			if options.debug {
				fmt.Println("[SOURCE] <<<\n" + string(message))
			}

			msg := parseSignalingMessage(string(message))

			lock.Lock()
			defer lock.Unlock()

			if msg.method == "ERROR" {
				fmt.Println("Error: " + msg.params["error-message"])
				killProcess()
			} else if msg.method == "OFFER" {
				if !receivedOffer {
					receivedOffer = true

					// Create peer connection
					peerConnectionConfig := loadWebRTCConfig() // Load config
					peerConnection, err = api.NewPeerConnection(peerConnectionConfig)
					if err != nil {
						fmt.Println("Error: " + err.Error())
						return
					}

					// Track listener
					peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
						lock.Lock()
						defer lock.Unlock()

						if receivedVideoTrack {
							return // Already received the track
						}

						if remoteTrack.Kind() != webrtc.RTPCodecTypeVideo {
							return // Not a video track
						}

						receivedVideoTrack = true

						// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
						go func() {
							ticker := time.NewTicker(time.Second * 2)
							for range ticker.C {
								if rtcpErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(remoteTrack.SSRC())}}); rtcpErr != nil {
									fmt.Println(rtcpErr)
								}
							}
						}()

						// Create SDP file
						sdpFile := createForwardSDPFile(options.port)

						// Forward track to RTP for FFmpeg to process
						go forwardTrack(remoteTrack, options.port)

						// Run publishing process
						go runPublish(sdpFile, destination, destinationStreamId, PublishOptions{
							debug:       options.debug,
							ffmpeg:      options.ffmpeg,
							authToken:   options.authTokenDestination,
							videoFilter: options.videoFilter,
						})
					})

					// ICE Candidate handler
					peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
						lock.Lock()
						defer lock.Unlock()

						candidateMsg := SignalingMessage{
							method: "CANDIDATE",
							params: make(map[string]string),
							body:   "",
						}
						candidateMsg.params["Request-ID"] = "play01"
						candidateMsg.params["Stream-ID"] = sourceStreamId
						if i != nil {
							b, e := json.Marshal(i.ToJSON())
							if e != nil {
								fmt.Println("Error: " + e.Error())
							} else {
								candidateMsg.body = string(b)
							}
						}

						c.WriteMessage(websocket.TextMessage, []byte(candidateMsg.serialize()))
						if options.debug {
							fmt.Println("[SOURCE] >>>\n" + string(candidateMsg.serialize()))
						}
					})

					peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
						lock.Lock()
						defer lock.Unlock()

						if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
							fmt.Println("[SOURCE] WebRTC: Disconnected")
							killProcess()
						} else if state == webrtc.PeerConnectionStateConnected {
							fmt.Println("[SOURCE] WebRTC: Connected")
						}
					})

					// Set remote rescription

					sd := webrtc.SessionDescription{}

					err := json.Unmarshal([]byte(msg.body), &sd)

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					err = peerConnection.SetRemoteDescription(sd)

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					// Generate answer
					answer, err := peerConnection.CreateAnswer(nil)
					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					// Sets the LocalDescription, and starts our UDP listeners
					err = peerConnection.SetLocalDescription(answer)
					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					// Send ANSWER to the client

					answerJSON, e := json.Marshal(answer)

					if e != nil {
						fmt.Println("Error: " + err.Error())
					}

					answerMsg := SignalingMessage{
						method: "ANSWER",
						params: make(map[string]string),
						body:   string(answerJSON),
					}
					answerMsg.params["Request-ID"] = "play01"
					answerMsg.params["Stream-ID"] = sourceStreamId

					c.WriteMessage(websocket.TextMessage, []byte(answerMsg.serialize()))

					if options.debug {
						fmt.Println(">>>\n" + string(answerMsg.serialize()))
					}
				}
			} else if msg.method == "CANDIDATE" {
				if receivedOffer && msg.body != "" {
					candidate := webrtc.ICECandidateInit{}

					err := json.Unmarshal([]byte(msg.body), &candidate)

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					err = peerConnection.AddICECandidate(candidate)

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}
				}
			} else if msg.method == "CLOSE" {
				fmt.Println("[SOURCE] Connection closed by remote host.")
				killProcess()
			}
		}()
	}
}
