# fakeflac-go

[![GitHub Release](https://img.shields.io/github/v/release/drichline/fakeflac-go?color=blue)](https://github.com/drichline/fakeflac-go/releases/latest) [![License](https://img.shields.io/github/license/drichline/fakeflac-go)](LICENSE) [![Go Version](https://img.shields.io/github/go-mod/go-version/drichline/fakeflac-go)](https://golang.org) [![Go Reference](https://pkg.go.dev/badge/github.com/drichline/fakeflac-go.svg)](https://pkg.go.dev/github.com/drichline/fakeflac-go)

A command-line tool to detect "fake" lossless audio files and optionally plot their frequency distribution. Analysis of lossy files is also supported. 

`fakeflac-go` estimates the percentage of the audio spectrum effectively remaining after a bad transcode from a lossy source.

Algorithm ported from [mevdschee/fakeflac](https://github.com/mevdschee/fakeflac/) with some adjustments; scores may not match. This algorithm is known to produce false negatives, see [Limitations](#limitations). 

## Features

- Numeric audio spectrum quality scores
- Optional plotting of audio spectrum
- Multithreaded batch processing of multiple input files
- Many lossless and lossy audio formats supported
- Tunable frequency cutoff detection parameters
- Portable single binary
- Embedded [WASM ffmpeg build](https://codeberg.org/gruf/go-ffmpreg/)
- Ability to use system ffmpeg install

## Installation

### Source installation

`go install github.com/drichline/fakeflac-go@v0.1.0`

This will compile and install the binary to `$GOPATH/bin`, where `$GOPATH` defaults to `~/go` on Linux/MacOS and `%USERPROFILE%\go` on Windows. You may have to add this directory to your system's `PATH`.

### Binary installation

Untested prebuilt binaries for Linux x86/ARM, MacOS Intel/ARM, and Windows 10+ are provided on the [releases page](https://github.com/drichline/fakeflac-go/releases); use at your own risk. 

## Requirements

- Linux/UNIX, Windows 10+, or MacOS 10.15+
- x86_64, ARM v6/v7/64, or Apple Silicon CPU
- 1-4 GB of free RAM for most operations
- Source installation: Go >= v1.27.1

Note: ffmpeg is embedded in `fakeflac-go` and thus not required to build or run

* If ffmpeg is installed, `-ffmpeg` uses the system ffmpeg for significantly improved performance, see [Usage](#usage).


## Quick Start


`$ fakeflac-go fake.flac real.flac `

Output:

```
fake.flac: 73
real.flac: 100
```

<a name="usage"></a>
## Usage

`fakeflac-go` accepts a space-separated list of filenames, and will ignore unsupported (e.g. text) files. Both lossless (e.g. `.flac`, `.alac`, `.wav`) and lossy (e.g. `.mp3`) formats are supported. For each file, `fakeflac-go` will output a score of 0-100 to the terminal, with 100 being a "perfect flac." 

The resulting numeric score represents the percentage of "real" frequencies up to 22 kHz present in the input file. Note that scores are only an estimate and may not be accurate for some edge-cases and lossy encoders, see [Limitations](#limitations). 

`-plot` saves the spectrum plot of each file to the current directory, named after the input file with `.png` appended. 

`-threads` sets the maximum number of active Goroutine workers. Defaults to the number of logical CPU cores present. 

* Note that by default, `$GOMAXPROCS` limits the maximum number of __active__ workers to the number of logical CPU cores, regardless of the limit set with `-threads`.

`-ffmpeg` uses the system's ffmpeg instead of embedded ffmpeg, greatly improving performance. 

* Note that `ffmpeg` must be available in the system's `$PATH`

`-boxcardx`, `-diff`, `-dx`, and `-limit`: see [Tuning](#tuning)

```
Usage: fakeflac-go [OPTIONS] [FILE]
Options:
  -plot
        Enable spectrum plot output
  -threads int
        Limit number of concurrent processes
  -ffmpeg
        Use system-installed ffmpeg if available
  -boxcardx int
        Number of boxcar filter spectrum bins (default 500)
  -diff float
        Lowpass cutoff magnitude drop test limit (default 1.25)
  -dx int
        Lowpass cutoff test window size in Hz (default 441)
  -limit float
        Lowpass cutoff magnitude ratio test limit (default 1.1)
```

## Supported file types

All input files are resampled to a 44.1 kHz 16 bit PCM stream by ffmpeg, so that the frequency range is normalized to 22 kHz (the upper limit of human ears). 

```
".flac", ".wav", ".w64", ".aif", ".aiff", ".aifc", ".au", ".snd",
".mp3", ".mp2", ".aac", ".m4a", ".m4b", ".mp4", ".ac3", ".eac3",
".ogg", ".oga", ".opus", ".spx", ".mka", ".weba", ".webm",
".wma", ".ape", ".wv", ".tta", ".tak", ".shn", ".mpc",
".caf", ".amr", ".dts", ".voc", ".dsf", ".dff", ".alac"
```

<a name="limitations"></a>
## Known limitations

Similar to the original `fakeflac.py`, `fakeflac-go` tends to produce false negatives (i.e. incorrect scores of 100) for some lossy encodes. Specifically, encodes that have sufficiently low magnitude at mid-high frequencies relative to the noise floor, and lack a steep drop-off in magnitude at the cutoff point. This can occur when audio is badly transcoded several times, very poor quality, or naturally very quiet in the upper frequencies, e.g. piano music. 

Future versions of `fakeflac-go` may include improved cutoff detection tests that use e.g. variance to detect a lowpassed noise floor. 

<img width="768" height="384" alt="Spectra of true positive and false negatives" src="https://github.com/user-attachments/assets/b46a8af4-ddf8-41e5-bff8-277e01d8ea16" />

Left: Fake flac spectrum that produces a false negative (score 100)

Right: Fake flac spectrum that produces a true positive (score 70)

<a name="tuning"></a>
## Tuning

The default constants included in `fakeflac-go` are empirical and based on those used in the original `fakeflac.py`. Tuning of these parameters may improve detection of lossy transcodes. Note that a frequency cutoff is only detected when both the limit and drop tests are satisfied. 

Run `fakeflac-go -help` to see defaults 

- dx: Number of frequencies, in Hz, to span while applying the limit and drop tests
- diff: Minimum magnitude difference between frequencies `dx` Hz apart to trigger drop test
- limit: Maximum difference ratio allowed between magnitude at 22 kHz and current tested frequency to trigger limit test
- boxcardx: Number of bins the spectrum is divided into by the boxcar filter; lower numbers reduce noise and variation at the expense of a blocky "step response"

## Images

<img width="768" height="384" alt="Fake (left) and real (right) flac file spectra" src="https://github.com/user-attachments/assets/2cf86c83-f202-43e5-8fa1-0ac0970a5d47" />

`fakeflac-go`-generated spectrum plot of "fake" versus real flac

---

<img width="2288" height="1103" alt="Fake (left) and real (right) flac file spectrograms" src="https://github.com/user-attachments/assets/5b2398c3-97d4-481a-9b7d-004d23d10e58" />

Example spectrograms of "fake" and real flacs, generated using `sox`
