// FFMPEG

package main

import (
	"fmt"
	"os"
	"os/exec"

	child_process_manager "github.com/AgustinSRG/go-child-process-manager"
)

func runEncdingProcess(ffmpegBin string, source string, videoUDP string, videoFilter string, debug bool) {
	args := make([]string, 1)

	args[0] = ffmpegBin

	args = append(args, "-re")

	args = append(args, "-protocol_whitelist", "file,sdp,udp,rtp")

	// INPUT
	args = append(args, "-i", source)

	// VIDEO OPTIONS
	args = append(args,
		"-an",
		"-vcodec", "libvpx",
		"-cpu-used", "5",
		"-deadline", "1",
		"-g", "10",
		"-error-resilient", "1",
		"-auto-alt-ref", "1",
	)

	// VIDEO FILTER
	if videoFilter != "" {
		args = append(args,
			"-vf", videoFilter,
		)
	}

	// VIDEO DESTINATION
	args = append(args,
		"-f", "rtp", "rtp://"+videoUDP+"?pkt_size=1200",
	)

	cmd := exec.Command(ffmpegBin)
	cmd.Args = args

	if debug {
		cmd.Stderr = os.Stderr
		fmt.Println("Running command: " + cmd.String())
	}

	child_process_manager.ConfigureCommand(cmd)

	err := cmd.Start()

	if err != nil {
		fmt.Println("Error: ffmpeg program failed: " + err.Error())
		os.Exit(1)
	}

	child_process_manager.AddChildProcess(cmd.Process)

	err = cmd.Wait()

	if err != nil {
		fmt.Println("Error: ffmpeg program failed: " + err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
