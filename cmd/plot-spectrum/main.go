// plot-spectrum generates spectrogram images from rf-scanner CSV output
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"strings"
)

var (
	inputFile  = flag.String("i", "", "Input CSV file from rf-scanner")
	outputFile = flag.String("o", "spectrogram.png", "Output PNG file")
	vmin       = flag.Float64("vmin", -80, "Minimum RSSI for color scale (dBm)")
	vmax       = flag.Float64("vmax", -30, "Maximum RSSI for color scale (dBm)")
	height     = flag.Int("height", 0, "Output image height (0 = auto, one pixel per frame)")
	colormap   = flag.String("cmap", "viridis", "Colormap: viridis, plasma, inferno, magma, turbo, grayscale")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -i spectrum.csv [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate spectrogram PNG from rf-scanner CSV output\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -i spectrum.csv                    # Default output\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i spectrum.csv -o out.png -vmin -70 -vmax -40\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i spectrum.csv -cmap turbo        # Use turbo colormap\n", os.Args[0])
	}
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -i input file required")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Read CSV file
	file, err := os.Open(*inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read header
	if !scanner.Scan() {
		return fmt.Errorf("empty CSV file")
	}
	header := scanner.Text()
	cols := strings.Split(header, ",")
	if len(cols) < 2 {
		return fmt.Errorf("invalid header: need at least timestamp and one frequency column")
	}

	freqs := make([]float64, len(cols)-1)
	for i, col := range cols[1:] {
		f, err := strconv.ParseFloat(col, 64)
		if err != nil {
			return fmt.Errorf("invalid frequency in header column %d: %w", i+1, err)
		}
		freqs[i] = f
	}

	// Read data rows
	var rows [][]float64
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		// Skip timestamp, parse RSSI values
		rssi := make([]float64, len(parts)-1)
		for i, p := range parts[1:] {
			v, err := strconv.ParseFloat(p, 64)
			if err != nil {
				rssi[i] = *vmin // Default to minimum if parse fails
			} else {
				rssi[i] = v
			}
		}
		rows = append(rows, rssi)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading CSV: %w", err)
	}

	if len(rows) == 0 {
		return fmt.Errorf("no data rows in CSV")
	}

	fmt.Printf("Loaded %d frames, %d frequency bins\n", len(rows), len(freqs))
	fmt.Printf("Frequency range: %.3f - %.3f MHz\n", freqs[0], freqs[len(freqs)-1])

	// Determine image dimensions
	imgWidth := len(freqs)
	imgHeight := len(rows)
	if *height > 0 {
		imgHeight = *height
	}

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Get colormap function
	cmap := getColormap(*colormap)

	// Fill image (time goes top to bottom, frequency left to right)
	for y := 0; y < imgHeight; y++ {
		// Map y to row index (handle scaling if height was specified)
		rowIdx := y * len(rows) / imgHeight
		if rowIdx >= len(rows) {
			rowIdx = len(rows) - 1
		}
		row := rows[rowIdx]

		for x := 0; x < imgWidth && x < len(row); x++ {
			// Normalize RSSI to 0-1 range
			normalized := (row[x] - *vmin) / (*vmax - *vmin)
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1 {
				normalized = 1
			}

			img.Set(x, y, cmap(normalized))
		}
	}

	// Write PNG
	outFile, err := os.Create(*outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	fmt.Printf("Wrote %dx%d spectrogram to %s\n", imgWidth, imgHeight, *outputFile)
	fmt.Printf("Color scale: %.1f to %.1f dBm\n", *vmin, *vmax)

	return nil
}

// Colormap function type
type colormapFunc func(t float64) color.RGBA

func getColormap(name string) colormapFunc {
	switch name {
	case "plasma":
		return plasmaColormap
	case "inferno":
		return infernoColormap
	case "magma":
		return magmaColormap
	case "turbo":
		return turboColormap
	case "grayscale":
		return grayscaleColormap
	default:
		return viridisColormap
	}
}

// Grayscale colormap
func grayscaleColormap(t float64) color.RGBA {
	v := uint8(t * 255)
	return color.RGBA{v, v, v, 255}
}

// Viridis colormap (perceptually uniform, good for scientific data)
func viridisColormap(t float64) color.RGBA {
	// Simplified viridis approximation
	r := uint8(clamp((-0.0029*t*t*t+1.2284*t*t-0.2547*t+0.2873)*255, 0, 255))
	g := uint8(clamp((0.0168*t*t*t-0.5523*t*t+1.1519*t+0.0058)*255, 0, 255))
	b := uint8(clamp((0.4401*t*t*t-1.4066*t*t+0.6717*t+0.3314)*255, 0, 255))
	return color.RGBA{r, g, b, 255}
}

// Plasma colormap
func plasmaColormap(t float64) color.RGBA {
	r := uint8(clamp((0.0504*t*t*t+0.6232*t*t+0.2889*t+0.0508)*255, 0, 255))
	g := uint8(clamp((-0.7924*t*t*t+0.5765*t*t+0.4694*t+0.0153)*255, 0, 255))
	b := uint8(clamp((0.5285*t*t*t-1.6325*t*t+0.6374*t+0.5299)*255, 0, 255))
	return color.RGBA{r, g, b, 255}
}

// Inferno colormap
func infernoColormap(t float64) color.RGBA {
	r := uint8(clamp((-0.0265*t*t*t+1.0977*t*t+0.0672*t+0.0002)*255, 0, 255))
	g := uint8(clamp((-0.3830*t*t*t+0.8453*t*t+0.2168*t-0.0118)*255, 0, 255))
	b := uint8(clamp((1.6132*t*t*t-2.7129*t*t+0.7959*t+0.0141)*255, 0, 255))
	return color.RGBA{r, g, b, 255}
}

// Magma colormap
func magmaColormap(t float64) color.RGBA {
	r := uint8(clamp((-0.1580*t*t*t+1.1943*t*t+0.1068*t+0.0002)*255, 0, 255))
	g := uint8(clamp((-0.4399*t*t*t+0.6573*t*t+0.4716*t-0.0045)*255, 0, 255))
	b := uint8(clamp((0.8754*t*t*t-1.7820*t*t+0.5787*t+0.0154)*255, 0, 255))
	return color.RGBA{r, g, b, 255}
}

// Turbo colormap (high contrast, good for visualizing details)
func turboColormap(t float64) color.RGBA {
	// Turbo is rainbow-like but perceptually better than jet
	var r, g, b float64

	if t < 0.25 {
		r = 0.18995 + t*4*(0.50344-0.18995)
		g = 0.07176 + t*4*(0.32263-0.07176)
		b = 0.23217 + t*4*(0.72595-0.23217)
	} else if t < 0.5 {
		t2 := (t - 0.25) * 4
		r = 0.50344 + t2*(0.96096-0.50344)
		g = 0.32263 + t2*(0.73552-0.32263)
		b = 0.72595 + t2*(0.22168-0.72595)
	} else if t < 0.75 {
		t2 := (t - 0.5) * 4
		r = 0.96096 + t2*(0.94505-0.96096)
		g = 0.73552 + t2*(0.91272-0.73552)
		b = 0.22168 + t2*(0.09430-0.22168)
	} else {
		t2 := (t - 0.75) * 4
		r = 0.94505 + t2*(0.47960-0.94505)
		g = 0.91272 + t2*(0.01583-0.91272)
		b = 0.09430 + t2*(0.01055-0.09430)
	}

	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
