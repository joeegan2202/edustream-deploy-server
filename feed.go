package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func (f *Feed) initiateStream() error {
	streamCommand := ""
	// Change with OS:
	var path []byte
	var err error
	if runtime.GOOS != "windows" {
		path, err = exec.Command("/usr/bin/which", "ffmpeg").Output()
	} else {
		path = []byte("ffmpeg\n")
	}

	if err != nil {
		return fmt.Errorf("Could not find ffmpeg binary/executable! Error: %s", err.Error())
	}

	streamCommand += string(path[0 : len(path)-1])
	fmt.Printf("Path found: %s\n", streamCommand)

	f.streamCmd = exec.Command(streamCommand, "-i", f.address, "-hls_time", hlsTime, "-hls_list_size", "20", "-hls_wrap", "20", "-codec", "copy", "-method", "PUT", fmt.Sprintf("http://127.0.0.1:%s/ingest/%s/stream.m3u8", os.Getenv("PORT"), f.id))
	fmt.Println(f.streamCmd.String())
	go func() {
		f.streamCmd.Run()
		index := -1
		for i, feed := range feeds {
			if feed.id == f.id {
				index = i
				break
			}
		}
		if index == -1 {
			return
		}
		feeds[len(feeds)-1], feeds[index] = feeds[index], feeds[len(feeds)-1]
		feeds = feeds[:len(feeds)-1] // Magic code to delete this feed from the list of feeds
		fmt.Printf("Stream is stopping: %s\n", f.streamCmd.String())
		sendStatus(f.id, 0)
	}()

	return nil
}
