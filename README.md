# fakeflac-go

A command-line tool to detect "fake" lossless audio files and optionally plot their frequency distribution. Analysis of lossy files is also supported. 

`fakeflac-go` estimates the percentage of the audio spectrum effectively remaining after a bad transcode from a lossy source

Algorithm ported from [mevdschee/fakeflac](https://github.com/mevdschee/fakeflac/) with some adjustments

# Features

- Numeric audio spectrum quality scores
- Optional plotting of audio spectrum
- Multithreaded batch processing of multiple input files
- Many lossless and lossy audio formats supported
- Tunable frequency cutoff detection parameters
- Portable single binary
- Embedded WASM ffmpeg build

# Requirements

- Go >= v1.21

Note: ffmpeg is embedded in `fakeflac-go` and thus not required

# Installation

`go install github.com/drichline/fakeflac-go@v0.0.1`

# Quick Start

```
$ fakeflac-go fake.flac real.flac 
fake.flac: 73
real.flac: 100
```
# Usage

`fakeflac-go` accepts a space-separated list of filenames, and will ignore unsupported (e.g. text) files. Both lossless (e.g. `.flac`, `.alac`, `.wav`) and lossy (e.g. `.mp3`) formats are supported. For each file, `fakeflac-go` will output a score of 0-100 to the terminal, with 100 being a perfect flac. 

The resulting numeric score represents the percentage of "real" frequencies up to 22 kHz present in the input file. 

`-plot` saves the spectrum plot of each file to the current directory. 

`-boxcarddx`, `-diff`, `-dx`, and `-limit` override the default empirical constants used for filtering the spectrum and detecting missing high frequencies

```

Usage: fakeflac-go [OPTIONS] [FILE]
Options:
  -plot
    	Enable spectrum plot output
  -boxcardx int
    	Boxcar filter window size (default 500) (default 500)
  -diff float
    	Lowpass cutoff magnitude drop test limit (default 1.25) (default 1.25)
  -dx int
    	Lowpass cutoff  (default 441)
  -limit float
    	Lowpass cutoff magnitude ratio test limit (default 1.1) (default 1.1)
```

## Supported file extensions

```
".flac", ".wav", ".w64", ".aif", ".aiff", ".aifc", ".au", ".snd",
".mp3", ".mp2", ".aac", ".m4a", ".m4b", ".mp4", ".ac3", ".eac3",
".ogg", ".oga", ".opus", ".spx", ".mka", ".weba", ".webm",
".wma", ".ape", ".wv", ".tta", ".tak", ".shn", ".mpc",
".caf", ".amr", ".dts", ".voc", ".dsf", ".dff", ".alac"
```

# Screenshots

<img width="768" height="384" alt="realfake" src="https://github.com/user-attachments/assets/2cf86c83-f202-43e5-8fa1-0ac0970a5d47" />

`fakeflac-go` plot of "fake" versus real flac

---

<img width="2288" height="1103" alt="realfakespec" src="https://github.com/user-attachments/assets/5b2398c3-97d4-481a-9b7d-004d23d10e58" />

Example spectrograms of "fake" and real flacs, generated using `sox`
