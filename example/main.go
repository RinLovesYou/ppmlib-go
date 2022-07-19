package main

import (
	"fmt"
	"image"
	"image/gif"
	"os"
	"time"

	"github.com/RinLovesYou/ppmlib-go"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

// func main() {
// 	ppm, err := ppmlib.ReadFile("bokeh.ppm")
// 	if err != nil {
// 		panic(err)
// 	}

// 	images := make([]*image.Paletted, ppm.FrameCount)
// 	for i := uint16(0); i < ppm.FrameCount; i++ {
// 		images[i] = ppm.Frames[i].GetImage()
// 	}

// 	buf, err := os.Create("bokeh.gif")
// 	if err != nil {
// 		panic(err)
// 	}

// 	timings := make([]int, ppm.FrameCount)

// 	for i := uint16(0); i < ppm.FrameCount; i++ {
// 		timings[i] = int(ppm.Framerate / 10000)
// 	}

// 	gif.EncodeAll(buf, &gif.GIF{
// 		Image: images,
// 		Delay: timings,
// 	})

// 	buf.Close()

// 	video, err := os.Open("bokeh.gif")
// 	if err != nil {
// 		panic(err)
// 	}

// 	audio, err := os.Create("bokeh.wav")
// 	if err != nil {
// 		panic(err)
// 	}

// 	ppm.Audio.Export(audio, ppm, 32768)

// 	audio.Close()
// 	ffmpeg_go.Input("pipe:0", ffmpeg_go.KwArgs{"format": "gif_pipe"}).WithInput(video).Output("bokeh.mp4").OverWriteOutput().ErrorToStdOut().Run()
// }

func main() {

	now := time.Now()

	name := "output"
	namePPM := fmt.Sprintf("%s.ppm", name)
	nameGIF := fmt.Sprintf("%s.gif", name)
	nameMP4 := fmt.Sprintf("%s.mp4", name)
	nameWAV := fmt.Sprintf("%s.wav", name)

	ppm, err := ppmlib.ReadFile(namePPM)
	if err != nil {
		panic(err)
	}

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

	for i := uint16(0); i < ppm.FrameCount; i++ {

	}

	gif.EncodeAll(gifFile, &gif.GIF{
		Image: images,
		Delay: timings,
	})

	audioFile, err := os.Create(nameWAV)
	if err != nil {
		panic(err)
	}

	defer audioFile.Close()

	ppm.Audio.Export(audioFile, ppm, 32768)

	var files_stream []*ffmpeg_go.Stream

	files_stream = append(files_stream, ffmpeg_go.Input(nameGIF, ffmpeg_go.KwArgs{"r": fmt.Sprintf("%.1f", ppm.Framerate)}).Video())
	files_stream = append(files_stream, ffmpeg_go.Input(nameWAV).Audio())

	err = ffmpeg_go.Concat(files_stream, ffmpeg_go.KwArgs{"v": 1, "a": 1}).
		Output(nameMP4, ffmpeg_go.KwArgs{
			"pix_fmt": "yuv420p",
			"c:v":     "libx264",
			"c:a":     "aac",
		}).
		OverWriteOutput().
		Run()

	if err != nil {
		panic(err)
	}

	audioFile.Close()
	gifFile.Close()

	os.Remove(nameGIF)
	os.Remove(nameWAV)

	elapsed := time.Since(now)
	fmt.Printf("parsing & encoding %s took %.1fs!\n", name, elapsed.Seconds())
}
