// Code to publish the filtered video track

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type PublishOptions struct {
	debug       bool
	ffmpeg      string
	authToken   string
	videoFilter string
}

func runPublish(source string, destination url.URL, streamId string, options PublishOptions) {
	// Create UDP listener
	listenerVideo, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		panic(err)
	}

	if options.debug {
		fmt.Println("UDP Listener openned for video: " + fmt.Sprint(listenerVideo.LocalAddr().String()))
	}

	// Create tracks

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		panic(err)
	}

	// Pipe tracks and start FFMPEG
	go pipeTrack(listenerVideo, videoTrack)
	go runEncdingProcess(options.ffmpeg, source, listenerVideo.LocalAddr().String(), options.videoFilter, options.debug)

	// Create peer connection
	peerConnectionConfig := loadWebRTCConfig() // Load config
	peerConnection, err := webrtc.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		return
	}

	// Mutex
	lock := sync.Mutex{}

	// Connect to websocket
	if options.debug {
		fmt.Println("Connecting to " + destination.String())
	}
	c, _, err := websocket.DefaultDialer.Dial(destination.String(), nil)
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
				fmt.Println("[DESTINATION] >>>\n" + string(heartbeatMessage.serialize()))
			}

			if sendErr != nil {
				return
			}
		}
	}()

	// Send publish message
	pubMsg := SignalingMessage{
		method: "PUBLISH",
		params: make(map[string]string),
		body:   "",
	}
	pubMsg.params["Request-ID"] = "pub01"
	pubMsg.params["Stream-ID"] = streamId
	pubMsg.params["Stream-Type"] = "VIDEO"
	if options.authToken != "" {
		pubMsg.params["Auth"] = options.authToken
	}
	c.WriteMessage(websocket.TextMessage, []byte(pubMsg.serialize()))

	if options.debug {
		fmt.Println("[DESTINATION] >>>\n" + string(pubMsg.serialize()))
	}

	// ICE Candidate handler
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		lock.Lock()
		defer lock.Unlock()

		candidateMsg := SignalingMessage{
			method: "CANDIDATE",
			params: make(map[string]string),
			body:   "",
		}
		candidateMsg.params["Request-ID"] = "pub01"
		candidateMsg.params["Stream-ID"] = streamId
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
			fmt.Println("[DESTINATION] >>>\n" + string(candidateMsg.serialize()))
		}
	})

	// Connection status handler
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		lock.Lock()
		defer lock.Unlock()

		if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
			fmt.Println("[DESTINATION] WebRTC: Disconnected")
		} else if state == webrtc.PeerConnectionStateConnected {
			fmt.Println("[DESTINATION] WebRTC: Connected")
		}
	})

	receivedOffer := false
	closed := false

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
				fmt.Println("<<<\n" + string(message))
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

					// Add tracks

					videoSender, err := peerConnection.AddTrack(videoTrack)
					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					go readPacketsFromRTPSender(videoSender)

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
					answerMsg.params["Request-ID"] = "pub01"
					answerMsg.params["Stream-ID"] = streamId

					c.WriteMessage(websocket.TextMessage, []byte(answerMsg.serialize()))

					if options.debug {
						fmt.Println("[DESTINATION] >>>\n" + string(answerMsg.serialize()))
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
				fmt.Println("[DESTINATION] Connection closed by remote host.")
				killProcess()
			}
		}()
	}
}
