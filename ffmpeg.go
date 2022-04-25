// FFMPEG

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runEncdingProcess(ffmpegBin string, source string, videoUDP string, debug bool) {
	args := make([]string, 1)

	args[0] = ffmpegBin

	// INPUT
	args = append(args, "-i", source)

	// VIDEO
	args = append(args,
		"-an",
		"-vcodec", "libvpx",
		"-cpu-used", "5",
		"-deadline", "1",
		"-g", "10",
		"-error-resilient", "1",
		"-auto-alt-ref", "1",
		"-f", "rtp", "rtp://"+videoUDP+"?pkt_size=1200",
	)

	cmd := exec.Command(ffmpegBin)
	cmd.Args = args

	if debug {
		cmd.Stderr = os.Stderr
		fmt.Println("Running command: " + cmd.String())
	}

	err := cmd.Run()

	if err != nil {
		fmt.Println("Error: ffmpeg program failed: " + err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
