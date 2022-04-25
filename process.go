// Video filtering script

package main

import "net/url"

type ProcessOptions struct {
	debug                bool
	ffmpeg               string
	authTokenSource      string
	authTokenDestination string
}

func runProcess(source url.URL, sourceStreamId string, destination url.URL, destinationStreamId string, options ProcessOptions) {
}
