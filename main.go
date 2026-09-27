package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"image/color"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/go-fft/fft"
	"github.com/tetratelabs/wazero"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const sampleRate int = 44100
const maxtime int = 30

func main() {
	spectrum := transform(convert(os.Args[1]))
	// pcmSamples := convert("/mnt/slab/media/music/Chris Hadfield - Space Sessions Songs from a Tin Can (2015) [Flac]/01 - Big Smoke.flac")

	points := make(plotter.XYs, len(spectrum))
	for x, y := range spectrum {
		points[x].X = float64(x)
		points[x].Y = y
	}

	s, err := plotter.NewScatter(points)
	if err != nil {
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
		Radius: 0.5,
		Shape:  draw.CircleGlyph{},
	}
	p.Add(s)

	err = p.Save(4*vg.Inch, 4*vg.Inch, "points.png")
	if err != nil {
		panic(err)
	}

	// var test bytes.Buffer
	// err := binary.Write(&test, binary.LittleEndian, pcmSamples)
	// if err != nil {
	// 	panic(err)
	// }

	// err = os.WriteFile(filepath.Join(os.TempDir(), "test.pcm"), test.Bytes(), 0644)
	// if err != nil {
	// 	panic(err)
	// }
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

func normalizeSpectrum(spectrum []float64) (normSpectrum []float64) {
	seconds := float64(min(len(spectrum)*2/sampleRate, maxtime)) // spectrum is only 0:samplerate/2 long, but contains 30s of summed spectra
	normSpectrum = make([]float64, len(spectrum))

	for i, item := range spectrum {
		normSpectrum[i] = item / seconds              // Average each magnitude over sample time
		normSpectrum[i] = math.Log10(normSpectrum[i]) // Normalize each magnitude
	}
	normSpectrum = boxcar(normSpectrum) // Apply boxcar filter

	return normSpectrum
}

// Boxcar moving average filter, so as to keep length(spectrum) == samplerate / 2
func boxcar(spectrum []float64) (avgSpectrum []float64) {
	window := sampleRate / 100
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

func transform(pcmSamples []int16) (spectrum []float64) {
	window := fft.Hann(sampleRate)
	audio := int2float(pcmSamples)
	seconds := min(len(pcmSamples)/sampleRate, maxtime)
	spectrum = make([]float64, sampleRate)
	audioSecond := make([]float64, sampleRate)

	for t := 0; t < seconds; t++ {
		floats.MulTo(audioSecond, window, audio[t*sampleRate:(t+1)*sampleRate])
		spectrumSecond := fft.FFTReal(audioSecond)
		spectrum = addSpectrum(spectrumSecond, spectrum)
	}

	spectrum = spectrum[0 : sampleRate/2]
	spectrum = normalizeSpectrum(spectrum)
	return spectrum
}

func convert(inputfile string) (pcmSamples []int16) {
	ctx, cncl := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cncl()

	// ext := filepath.Ext(inputfile)
	// outputFile := filepath.Join(os.TempDir(), filepath.Base(strings.TrimSuffix(inputfile, ext)))
	// outputFile += ".pcm"

	var pcmBuffer bytes.Buffer
	pcmReceiver := bufio.NewWriter(&pcmBuffer)

	_, err := ffmpreg.Run(ctx, wasm.Args{
		Name: "ffmpeg",
		// Stdin:  os.Stdin,
		// Stdout: os.Stdout,
		Stdout: pcmReceiver,
		Stderr: os.Stderr,
		Args: []string{"-i", inputfile,
			"-vn",
			"-ar", "44100",
			"-ac", "1",
			"-acodec", "pcm_s16le",
			"-f", "s16le",
			"pipe:1"},
		Config: func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
			for _, kv := range os.Environ() {
				i := strings.IndexByte(kv, '=')
				cfg = cfg.WithEnv(kv[:i], kv[i+1:])
			}
			fscfg := wazero.NewFSConfig()
			fscfg = fscfg.WithDirMount("/", "/")
			return cfg.WithFSConfig(fscfg)
		},
	})
	if err != nil {
		panic(err)
	}

	pcmSamples = make([]int16, len(pcmBuffer.Bytes())/2)
	for i := range pcmSamples {
		pcmSamples[i] = int16(binary.LittleEndian.Uint16(pcmBuffer.Bytes()[i*2:]))
	}
	return pcmSamples
}
