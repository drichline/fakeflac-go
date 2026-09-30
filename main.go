package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/go-fft/fft"
	"github.com/tetratelabs/wazero"
	"golang.org/x/sync/errgroup"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// Sample rate used for PCM waveform conversion
const sampleRate int = 44100 // Note: Frequency range is half of sample rate
const maxtime int = 30

// Emperical constant defaults
const defaultdx int = sampleRate / 100
const defaultdiff float64 = 1.25
const defaultlimit float64 = 1.1
const defaultboxcardx int = 500

func main() {
	// Set up CLI flags
	var plotFlag = flag.Bool("plot", false, "Enable spectrum plot output")
	var threadFlag = flag.Int("threads", runtime.NumCPU(), "Limit number of concurrent processes")
	var ffmpegFlag = flag.Bool("ffmpeg", false, "Use native ffmpeg if available")
	var dxFlag = flag.Int("dx", defaultdx, "Lowpass cutoff test window size in Hz")
	var diffFlag = flag.Float64("diff", defaultdiff, "Lowpass cutoff magnitude drop test limit")
	var limitFlag = flag.Float64("limit", defaultlimit, "Lowpass cutoff magnitude ratio test limit")
	var boxcardxFlag = flag.Int("boxcardx", defaultboxcardx, "Number of boxcar filter spectrum bins")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fakeflac-go [OPTIONS] [FILE]\nOptions:\n")
		flag.PrintDefaults()
	}
	// Parse CLI arguments
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fakeflac-go [OPTIONS] [FILE]\nTry 'fakeflac-go -help' for more information.\n")
		os.Exit(1)
	}
	boxcarWindow := (sampleRate / *boxcardxFlag) / 2
	files := flag.Args()
	files = matchExt(files) // Discard non-audio files

	// Limit goroutines to # of CPU threads
	var workerPool errgroup.Group
	workerPool.SetLimit(*threadFlag)
	var printLock sync.Mutex // Block to prevent concurrent prints

	for _, file := range files {
		workerPool.Go(func() error {
			file, err := filepath.Abs(file)
			if err != nil {
				println("Error getting working directory")
				panic(err)
			}

			spectrum := normalizeSpectrum(transform(convert(file, *ffmpegFlag)), boxcarWindow)
			cutoff := findCutoff(spectrum, *dxFlag, *diffFlag, *limitFlag)

			if *plotFlag {
				plotSpec(spectrum, filepath.Base(file))
			}

			// Return frequency cutoff percentage
			printLock.Lock()
			fmt.Printf("%s: %d\n", filepath.Base(file), cutoff)
			printLock.Unlock()

			return nil
		})
	}
	if err := workerPool.Wait(); err != nil {
		panic(err)
	}
}

func matchExt(files []string) (audioFiles []string) {
	allowedExt := []string{
		".flac", ".wav", ".w64", ".aif", ".aiff", ".aifc", ".au", ".snd",
		".mp3", ".mp2", ".aac", ".m4a", ".m4b", ".mp4", ".ac3", ".eac3",
		".ogg", ".oga", ".opus", ".spx", ".mka", ".weba", ".webm",
		".wma", ".ape", ".wv", ".tta", ".tak", ".shn", ".mpc",
		".caf", ".amr", ".dts", ".voc", ".dsf", ".dff", ".alac",
	}
	audioFiles = files
	audioFiles = slices.DeleteFunc(audioFiles, func(name string) bool {
		return !slices.Contains(allowedExt, strings.ToLower(filepath.Ext(name)))
	})

	if len(audioFiles) == 0 {
		println("Supported file types: " + strings.Join(allowedExt, " "))
		os.Exit(1)
	}

	return audioFiles
}

func plotSpec(spectrum []float64, filename string) {
	points := make(plotter.XYs, len(spectrum))
	for x, y := range spectrum {
		points[x].X = float64(x)
		points[x].Y = y
	}

	s, err := plotter.NewScatter(points)
	if err != nil {
		println("Failed to construct scatter plot")
		panic(err)
	}

	p := plot.New()
	p.Title.Text = "Spectrum"
	p.X.Label.Text = "Frequency"
	p.Y.Label.Text = "Magnitude"
	p.X.Min = 0
	p.X.Max = float64(sampleRate / 2)
	p.Y.Min = 0
	p.Y.Max = 10

	s.GlyphStyle = draw.GlyphStyle{
		Color:  color.Black,
		Radius: 0.2,
		Shape:  draw.CircleGlyph{},
	}
	p.Add(s)

	err = p.Save(4*vg.Inch, 4*vg.Inch, filename+".png")
	if err != nil {
		println("Failed to save plot")
		panic(err)
	}

}

func findCutoff(spectrum []float64, dx int, diff float64, limit float64) (cutoff int) {
	end := len(spectrum) - 1
	for i := end; i >= dx; i-- {
		// Ratio test AND drop test must trigger to declare cutoff found
		if spectrum[i]/spectrum[end] < limit && spectrum[i-dx]-spectrum[i] > diff {
			end = i - dx
			break
		}
	}
	cutoff = (end + 1) * 100 / len(spectrum)
	return cutoff
}

func int2float(intSlice []int16) (floatSlice []float64) {
	floatSlice = make([]float64, len(intSlice))
	for i, item := range intSlice {
		floatSlice[i] = float64(item)
	}
	return floatSlice
}

func addSpectrum(spectrumSecond []complex128, spectrum []float64) (spectrumSum []float64) {
	spectrumSum = make([]float64, len(spectrumSecond))
	for i, item := range spectrumSecond {
		spectrumSum[i] = spectrum[i] + math.Abs(real(item))
	}
	return spectrumSum
}

// Boxcar moving average filter, so as to keep length(spectrum) == samplerate / 2
func boxcar(spectrum []float64, window int) (avgSpectrum []float64) {
	n := len(spectrum)
	avgSpectrum = make([]float64, n)

	for start := 0; start < n; start += window { //
		end := start + window
		if end > n { // Don't overrun end of spectrum
			end = n
		}

		var sum float64
		for i := start; i < end; i++ { // Running sum of spectrum magnitudes within window
			sum += spectrum[i]
		}
		avg := sum / float64(end-start) // Average of magnitudes within window

		for i := start; i < end; i++ { // Fill window with average of window
			avgSpectrum[i] = avg
		}
	}
	return avgSpectrum
}

func normalizeSpectrum(spectrum []float64, boxcarWindow int) (normSpectrum []float64) {
	seconds := float64(min(len(spectrum)*2/sampleRate, maxtime)) // spectrum is only 0:samplerate/2 long, but contains 30s of summed spectra
	normSpectrum = make([]float64, len(spectrum))

	for i, item := range spectrum {
		normSpectrum[i] = item / seconds              // Average each magnitude over sample time
		normSpectrum[i] = math.Log10(normSpectrum[i]) // Normalize each magnitude
	}
	normSpectrum = boxcar(normSpectrum, boxcarWindow) // Apply boxcar filter
	return normSpectrum
}

func transform(pcmSamples []int16) (spectrum []float64) {
	window := fft.Hann(sampleRate)
	audio := int2float(pcmSamples)
	seconds := min(len(pcmSamples)/sampleRate, maxtime)
	spectrum = make([]float64, sampleRate)
	audioSecond := make([]float64, sampleRate)

	for t := range seconds {
		floats.MulTo(audioSecond, window, audio[t*sampleRate:(t+1)*sampleRate])
		spectrumSecond := fft.FFTReal(audioSecond)
		spectrum = addSpectrum(spectrumSecond, spectrum)
	}

	spectrum = spectrum[0 : sampleRate/2]
	return spectrum
}

func convert(inputfile string, ffmpegFlag bool) (pcmSamples []int16) {
	// Buffer to intercept PCM stream from ffmpreg output
	var pcmBuffer bytes.Buffer
	pcmReceiver := bufio.NewWriter(&pcmBuffer)

	// ffmpeg arguments
	args := []string{"-i", inputfile,
		"-vn", // Discard video streams
		"-ar", strconv.Itoa(sampleRate),
		"-ac", "1", // Mono non-interleaved PCM
		"-acodec", "pcm_s16le", // Raw 16 bit PCM stream
		"-f", "s16le", // No container; raw bytes
		"pipe:1"} // Output PCM stream to buffer

	// Embedded WASM ffmpeg
	embedded := func() {
		ctx := context.Background()
		_, err := ffmpreg.Run(ctx, wasm.Args{
			Name: "ffmpeg",
			// Stdin:  os.Stdin,
			// Stdout: os.Stdout,
			Stdout: pcmReceiver,
			// Stderr: os.Stderr,
			Args: args,
			Config: func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
				// Unused ffmpeg env var handling
				// for _, kv := range os.Environ() {
				// 	i := strings.IndexByte(kv, '=')
				// 	cfg = cfg.WithEnv(kv[:i], kv[i+1:])
				// }
				fscfg := wazero.NewFSConfig()
				fscfg = fscfg.WithReadOnlyDirMount("/", "/")
				return cfg.WithFSConfig(fscfg)
			},
		})
		if err != nil {
			println("Embedded ffmpeg PCM encoding failed on file " + filepath.Base(inputfile))
			panic(err)
		}
	}

	// Shell out to system ffmpeg
	native := func() {
		ffmpeg := exec.Command("ffmpeg", args...)
		ffmpeg.Stdout = &pcmBuffer
		err := ffmpeg.Run()
		if err != nil {
			println("Native ffmpeg PCM encoding failed on file " + filepath.Base(inputfile))
			panic(err)
		}
	}

	// Determine whether ffmpeg is available
	ffmpegAvailable := func() bool {
		_, err := exec.LookPath("ffmpeg")
		return err == nil
	}

	// Default to embedded ffmpeg
	if ffmpegFlag && ffmpegAvailable() {
		native()
	} else if ffmpegFlag {
		println("ffmpeg not detected; falling back to embedded ffmpeg")
		embedded()
	} else {
		embedded()
	}

	// Convert raw PCM stream into a slice of 16-bit integer samples
	pcmSamples = make([]int16, len(pcmBuffer.Bytes())/2)
	for i := range pcmSamples {
		pcmSamples[i] = int16(binary.LittleEndian.Uint16(pcmBuffer.Bytes()[i*2:]))
	}
	return pcmSamples
}
