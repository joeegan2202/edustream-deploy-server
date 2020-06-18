package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func (f *Feed) initiateStream() error {
  streamCommand := ""
  // Change with OS:
  var path []byte
  var err error
  if runtime.GOOS == "windows" {
    path, err = exec.Command("where ffmpeg").Output()
  } else {
    path, err = exec.Command("/usr/bin/which", "ffmpeg").Output()
  }

  syscall.Umask(0)
  os.Mkdir(fmt.Sprintf("streams/%s", f.id), 0755)

  if err != nil {
    return fmt.Errorf("Could not find ffmpeg binary/executable! Error: %s", err.Error())
  }

  streamCommand += string(path[0:len(path)-1])
  fmt.Printf("Path found: %s\n", streamCommand)

  f.streamCmd = exec.Command(streamCommand, "-i", f.address, "-hls_time", "3", "-hls_wrap", "10", "-codec", "copy", fmt.Sprintf("streams/%s/stream.m3u8", f.id))
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
  }()

  return nil
}
