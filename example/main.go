package main

import (
	"fmt"
	"image"
	"image/gif"
	"os"
	"time"

	"github.com/Clinet/ffgoconv"
	"github.com/RinLovesYou/ppmlib-go"
)

func main() {
	if len(os.Args) < 2 {
		panic("you must specify a flipnote! exclude the .ppm for now or it'll think .ppm.ppm and .ppm.mp4")
	}

	name := os.Args[1]
	namePPM := fmt.Sprintf("%s.ppm", name)
	nameGIF := fmt.Sprintf("%s.gif", name)
	nameWAV := fmt.Sprintf("%s.wav", name)
	nameMP4 := fmt.Sprintf("%s.mp4", name)

	os.Remove(nameGIF)
	os.Remove(nameWAV)
	os.Remove(nameMP4)

	fmt.Println("Starting...")
	timeStart := time.Now()

	timePPM := time.Now()
	ppm, err := ppmlib.ReadFile(namePPM)
	if err != nil {
		panic(err)
	}
	fmt.Printf("PPM: Parsed %s in %dms!\n", name, time.Since(timePPM).Milliseconds())

	timeGIF := time.Now()
	images := make([]*image.Paletted, ppm.FrameCount)
	for i := uint16(0); i < ppm.FrameCount; i++ {
		images[i] = ppm.Frames[i].GetImage()
	}

	gifFile, err := os.Create(nameGIF)
	if err != nil {
		panic(err)
	}
	defer gifFile.Close()

	timings := make([]int, ppm.FrameCount)

	gif.EncodeAll(gifFile, &gif.GIF{
		Image: images,
		Delay: timings,
	})
	fmt.Printf("GIF: Encoded %s in %dms!\n", name, time.Since(timeGIF).Milliseconds())

	timeWAV := time.Now()
	audioFile, err := os.Create(nameWAV)
	if err != nil {
		panic(err)
	}
	defer audioFile.Close()

	ppm.Audio.Export(audioFile, ppm, 32768)
	fmt.Printf("WAV: Encoded %s in %dms!\n", name, time.Since(timeWAV).Milliseconds())

	timeMP4 := time.Now()
	ffmpeg, err := ffgoconv.NewFFmpeg(name, []string{"-hide_banner", "-stats",
		"-r", fmt.Sprintf("%.1f", ppm.Framerate),
		"-hwaccel", "auto",
		"-i", nameGIF,
		"-i", nameWAV,
		nameMP4,
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-threads", "0",
	})
	if err != nil {
		panic(err)
	}

	for {
		if !ffmpeg.IsRunning() {
			break
		}
		time.Sleep(time.Millisecond * 1)
	}
	if ffmpeg.Err() != nil {
		panic(ffmpeg.Err())
	}
	fmt.Printf("MP4: Encoded %s in %dms!\n", name, time.Since(timeMP4).Milliseconds())

	fmt.Printf("Parsing and encoding %s took a total of %dms!\n", name, time.Since(timeStart).Milliseconds())

	//ppm.Save("copy.ppm")
}
