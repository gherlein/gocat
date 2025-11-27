# GUI Options for GoCat

## RFCat GUI Functionality Overview

### Spectrum Analyzer (ccspecan.py)

RFCat's primary graphical component is a real-time spectrum analyzer that visualizes RF signal strength across frequency ranges. The implementation uses **PySide2** (Qt for Python) and provides:

#### Core Visualization Features
- **Real-time spectrum display**: Continuously plots frequency vs. signal strength (RSSI/dBm)
- **Persistence tracking**: Maintains history of 350 frames to show maximum signal envelope over time
- **Dual-layer rendering**:
  - Graph layer: Frequency/power data with configurable trail persistence
  - Reticle layer: Grid overlay with frequency and dBm scale markings
- **Color-coded display**:
  - White: Current signal path
  - Green: Maximum signal envelope (peak hold)
  - Crosshairs: Interactive measurement cursors

#### Interactive Features
- **Mouse-based cursors**: Left/right click to place frequency/power markers
- **Frequency difference calculations**: Shows delta between two marked frequencies
- **Keyboard navigation**: Arrow keys to adjust frequency range and step size
- **Pan/zoom**: Dynamic rescaling without halting data acquisition
- **Toggle overlays**: Hide/show measurement markers
- **Help system**: Built-in command reference

#### Technical Architecture
- **Threading**: Separate `SpecanThread` for RF data acquisition to keep UI responsive
- **Data structures**: NumPy arrays for efficient storage of historical RSSI values
- **Coordinate mapping**: Transformation functions convert Hz/dBm to screen pixels
- **Rendering APIs**:
  - `QPainter` for all drawing operations
  - `QPixmap` for off-screen rendering buffers
  - `QPainterPath` for smooth curve plotting
  - Antialiasing enabled for visual quality

#### Data Flow
1. RF device continuously streams RSSI measurements
2. Background thread receives and calibrates values: `(value^0x80)/2 - 88`
3. Frame callback triggers UI update
4. `RenderArea` widget redraws spectrum with latest data
5. Historical frames maintained for persistence visualization

### Other GUI Components

**IMME Display**: RFCat includes firmware for the GirlTech IMME toy that provides a simple LCD display for frequency and packet information directly on the device hardware.

**Third-party GUIs**: Community projects like [rfcat-gui](https://github.com/Lupin3000/rfcat-gui) provide Tkinter-based configuration interfaces with input fields for RF parameters, though these are less mature than the spectrum analyzer.

## Go GUI Options for GoCat

### Option 1: Fyne + Custom Canvas

#### Description
[Fyne](https://fyne.io/) is a modern, cross-platform GUI toolkit written in pure Go, inspired by Material Design.

#### Pros
- **Pure Go**: No CGO dependencies, easy compilation
- **Cross-platform**: Windows, macOS, Linux, iOS, Android, WASM
- **GPU-accelerated**: Uses OpenGL for hardware rendering
- **Modern API**: Clean, idiomatic Go interface
- **Active development**: Well-maintained with regular releases
- **Built-in widgets**: Rich set of standard UI components
- **Custom drawing**: `canvas.Raster` and `fyne.CanvasObject` for custom graphics
- **Responsive layouts**: Automatic resizing and scaling

#### Cons
- **Custom plotting required**: No built-in spectrum/chart widgets
- **Canvas API**: More limited than full OpenGL access
- **Memory usage**: Can be higher for complex custom rendering

#### Implementation Approach
```go
// Use Fyne's canvas.Raster for real-time spectrum display
type SpectrumRenderer struct {
    data []float64
    // Historical frames for persistence
    frames [][]float64
}

func (s *SpectrumRenderer) Render(w, h int) image.Image {
    // Draw spectrum graph with persistence
    // Draw grid reticle
    // Draw cursors and labels
}
```

#### Best For
- Applications prioritizing ease of deployment and cross-platform support
- Projects wanting a complete GUI toolkit with custom visualization
- Teams familiar with Go but not graphics programming

### Option 2: Gio + Custom Graphics

#### Description
[Gio](https://gioui.org/) implements portable immediate-mode GUI programs in Go with a focus on performance and modern rendering.

#### Pros
- **Pure Go**: No CGO dependencies
- **Immediate mode**: Efficient rendering model similar to game engines
- **Vulkan/Metal/Direct3D**: Modern graphics APIs (not just OpenGL)
- **Low-level control**: More direct access to rendering pipeline
- **Efficient**: Designed for high-performance graphics
- **Flexible**: Custom layouts and drawing primitives
- **Small binaries**: Minimal overhead

#### Cons
- **Steeper learning curve**: Immediate-mode paradigm different from traditional GUI
- **Less documentation**: Smaller community than Fyne
- **Custom everything**: More code required for standard UI elements
- **API still evolving**: Some breaking changes between versions

#### Implementation Approach
```go
// Use Gio's paint operations for direct rendering
func drawSpectrum(ops *op.Ops, data []float64) {
    // Build paint operations for spectrum
    paint.ColorOp{Color: color.NRGBA{...}}.Add(ops)
    // Draw path for signal
    // Draw grid and labels
}
```

#### Best For
- Performance-critical applications requiring 60+ FPS
- Projects needing modern GPU features
- Developers comfortable with immediate-mode rendering
- Minimal binary size requirements

### Option 3: Go-GL + GLFW + Custom Framework

#### Description
Direct OpenGL programming using [go-gl](https://github.com/go-gl) bindings with [GLFW](https://www.glfw.org/) for window management.

#### Pros
- **Maximum control**: Full OpenGL API access
- **Best performance**: Direct GPU programming
- **Flexibility**: Can implement any visual effect
- **Industry standard**: OpenGL knowledge transfers to other domains
- **Mature bindings**: go-gl is well-established
- **Custom everything**: Total control over rendering pipeline

#### Cons
- **Requires CGO**: Complicates cross-compilation
- **Complexity**: Must implement all UI elements from scratch
- **Verbose**: More boilerplate code
- **Platform differences**: OpenGL versions vary by platform
- **No UI toolkit**: Buttons, text input, layouts all manual
- **Development time**: Significant effort to build GUI framework

#### Implementation Approach
```go
// Direct OpenGL rendering
func renderSpectrum(data []float64) {
    gl.Clear(gl.COLOR_BUFFER_BIT)

    // Upload data to GPU buffer
    gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
    gl.BufferData(gl.ARRAY_BUFFER, data, gl.DYNAMIC_DRAW)

    // Draw with shaders
    shader.Use()
    gl.DrawArrays(gl.LINE_STRIP, 0, len(data))
}
```

#### Best For
- Maximum performance requirements
- Projects with existing OpenGL expertise
- Highly specialized visualization needs
- Applications where GUI toolkit restrictions are limiting

### Option 4: Gonum/Plot + GUI Toolkit

#### Description
Use [gonum/plot](https://github.com/gonum/plot) for chart generation embedded in a Fyne or Gio window.

#### Pros
- **Plotting library included**: Pre-built chart types
- **Scientific focus**: Designed for data visualization
- **Publication quality**: Nice-looking default output
- **Pure Go**: No CGO required
- **Multiple backends**: PNG, SVG, PDF output
- **Good for static charts**: Excellent for report generation

#### Cons
- **Not real-time**: Designed for static image generation
- **Performance**: Regenerating full plots every frame is slow
- **Limited interactivity**: No built-in cursors or zoom
- **Workaround required**: Need to regenerate and redisplay images
- **Memory overhead**: Creating new images constantly

#### Implementation Approach
```go
// Generate plot and display as image
func updateSpectrum(data []float64) {
    p := plot.New()
    line, _ := plotter.NewLine(convertToXY(data))
    p.Add(line)

    // Save to buffer
    var buf bytes.Buffer
    p.WriterTo(4*vg.Inch, 3*vg.Inch, "png")

    // Display in Fyne Image widget
    img := canvas.NewImageFromReader(&buf, "spectrum")
}
```

#### Best For
- Occasional spectrum snapshots (not continuous streaming)
- Applications with mostly static visualization
- Projects prioritizing code simplicity over performance

### Option 5: Qt Bindings (therecipe/qt)

#### Description
Go bindings for the Qt framework, mirroring RFCat's PySide2 approach.

#### Pros
- **Feature parity**: Can replicate RFCat GUI closely
- **Mature framework**: Qt is industry-standard
- **Rich widgets**: Extensive built-in components
- **Custom painting**: QPainter available like Python version
- **Cross-platform**: Windows, macOS, Linux, Android, iOS
- **Familiar**: Direct translation from Python possible

#### Cons
- **CGO required**: Complex build process
- **Large dependencies**: Qt libraries must be installed
- **Project status**: therecipe/qt maintenance is uncertain
- **Build complexity**: Cross-compilation is difficult
- **Binary size**: Large executables due to Qt
- **Violates requirement #2**: "must be in go with no non-go Requirements"

#### Implementation Approach
```go
// Direct port from Python
type RenderArea struct {
    widgets.QWidget
    frames [][]float64
}

func (r *RenderArea) paintEvent(event *gui.QPaintEvent) {
    painter := gui.NewQPainter2(r)
    // Nearly identical code to Python version
}
```

#### Best For
- **NOT RECOMMENDED for GoCat** due to non-Go dependencies requirement
- Teams already using Qt in other parts of stack
- Direct Python-to-Go ports

## Recommendations

### Primary Recommendation: Fyne + Custom Canvas

**For GoCat's spectrum analyzer, Fyne with custom canvas rendering is the best choice.**

#### Rationale
1. **Meets all requirements**:
   - Pure Go, no non-Go dependencies ✓
   - Cross-platform with Linux priority ✓
   - Can be library or standalone ✓
   - Easy to use for experimentation ✓

2. **Balanced approach**:
   - Simpler than raw OpenGL but still performant
   - Provides UI toolkit for controls (buttons, sliders, text)
   - Custom canvas for spectrum visualization
   - Active community and good documentation

3. **Performance sufficient**:
   - GPU-accelerated rendering via OpenGL
   - Can handle 30-60 FPS spectrum updates
   - Efficient enough for RF visualization (not 3D gaming)

4. **Development velocity**:
   - Faster than building from scratch with OpenGL
   - Cleaner API than immediate-mode Gio
   - Good balance of control vs. convenience

#### Implementation Strategy

```go
package gocat

import (
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "image"
    "image/color"
)

// SpectrumAnalyzer provides real-time RF spectrum visualization
type SpectrumAnalyzer struct {
    device      *RFDevice
    raster      *canvas.Raster
    frames      [][]float64  // Historical data for persistence
    cursorLeft  float64      // Hz
    cursorRight float64      // Hz
    centerFreq  float64
    span        float64
}

func NewSpectrumAnalyzer(device *RFDevice) *SpectrumAnalyzer {
    sa := &SpectrumAnalyzer{
        device: device,
        frames: make([][]float64, 350), // Match RFCat persistence depth
    }

    sa.raster = canvas.NewRasterWithPixels(sa.render)
    return sa
}

func (sa *SpectrumAnalyzer) render(w, h int) color.Color {
    // Called for each pixel, return appropriate color
    // Draw spectrum line, persistence envelope, grid, cursors
}

func (sa *SpectrumAnalyzer) Show() {
    a := app.New()
    w := a.NewWindow("GoCat Spectrum Analyzer")

    // Create control panel
    freqLabel := widget.NewLabel("Center Frequency:")
    freqEntry := widget.NewEntry()
    spanSlider := widget.NewSlider(100e3, 10e6)

    controls := container.NewVBox(
        freqLabel,
        freqEntry,
        spanSlider,
    )

    // Layout with spectrum display and controls
    content := container.NewBorder(
        nil,    // top
        controls, // bottom
        nil,    // left
        nil,    // right
        sa.raster, // center
    )

    w.SetContent(content)

    // Start data acquisition goroutine
    go sa.acquireData()

    w.ShowAndRun()
}

func (sa *SpectrumAnalyzer) acquireData() {
    for {
        data := sa.device.GetSpectrumData()
        sa.addFrame(data)
        sa.raster.Refresh()
    }
}
```

### Secondary Recommendation: Gio (for performance-critical use)

If performance profiling shows Fyne cannot maintain desired frame rates (60+ FPS), or if binary size is critical, **Gio is the next best choice**.

Gio's immediate-mode rendering and modern GPU API support provide maximum performance while staying pure Go. The trade-off is more complex code and steeper learning curve.

### Alternative: Headless Mode + Web Interface

For scenarios where native GUI is not required:

```go
// Serve spectrum data via WebSocket
// Frontend uses HTML5 Canvas or WebGL for visualization
// Advantages: Remote access, multi-platform, familiar web tech
// Disadvantages: Requires browser, more complex architecture
```

This could be a complementary mode rather than replacement, similar to how RFCat has multiple interfaces (CLI, GUI, server).

## Implementation Considerations

### Data Acquisition Pattern

Use separate goroutines for RF data and GUI rendering:

```go
type SpectrumData struct {
    frequencies []float64
    rssi        []float64
    timestamp   time.Time
}

// RF acquisition goroutine
func (d *RFDevice) StreamSpectrum(ch chan<- SpectrumData) {
    for {
        data := d.scanFrequencies()
        ch <- data
    }
}

// GUI update goroutine
func (sa *SpectrumAnalyzer) updateLoop(ch <-chan SpectrumData) {
    for data := range ch {
        sa.updateDisplay(data)
    }
}
```

### Coordinate Transformations

Mirror RFCat's approach with helper functions:

```go
func (sa *SpectrumAnalyzer) hzToX(freq float64, width int) int {
    // Map frequency to pixel X coordinate
    normalized := (freq - sa.minFreq) / (sa.maxFreq - sa.minFreq)
    return int(normalized * float64(width))
}

func (sa *SpectrumAnalyzer) dbmToY(power float64, height int) int {
    // Map dBm to pixel Y coordinate (inverted)
    normalized := (power - sa.minDbm) / (sa.maxDbm - sa.minDbm)
    return height - int(normalized * float64(height))
}
```

### Persistence Implementation

Maintain circular buffer of historical frames:

```go
type PersistenceBuffer struct {
    frames [][]float64
    index  int
    size   int
}

func (pb *PersistenceBuffer) Add(frame []float64) {
    pb.frames[pb.index] = frame
    pb.index = (pb.index + 1) % pb.size
}

func (pb *PersistenceBuffer) MaxEnvelope() []float64 {
    if len(pb.frames) == 0 {
        return nil
    }

    maxValues := make([]float64, len(pb.frames[0]))
    for _, frame := range pb.frames {
        for i, val := range frame {
            if val > maxValues[i] {
                maxValues[i] = val
            }
        }
    }
    return maxValues
}
```

## Summary

For GoCat's spectrum analyzer functionality:

1. **Use Fyne** as primary GUI framework
2. **Implement custom canvas rendering** for spectrum display
3. **Follow RFCat's architecture** (separate acquisition thread, persistence buffer, coordinate mapping)
4. **Keep it optional** - spectrum analyzer should be importable library, not required
5. **Consider Gio** if performance testing shows need for optimization
6. **Avoid Qt bindings** - violates pure Go requirement
7. **Skip gonum/plot** - not suitable for real-time visualization

This approach provides the best balance of functionality, performance, maintainability, and adherence to project requirements.

## References

- [RFCat GitHub Repository](https://github.com/atlas0fd00m/rfcat)
- [RFCat Spectrum Analyzer Source](https://github.com/atlas0fd00m/rfcat/blob/master/rflib/ccspecan.py)
- [Lupin3000 RFCat GUI (Tkinter)](https://github.com/Lupin3000/rfcat-gui)
- [Fyne Framework](https://fyne.io/)
- [Gio Framework](https://gioui.org/)
- [go-gl OpenGL Bindings](https://github.com/go-gl)
- [gonum/plot Plotting Library](https://github.com/gonum/plot)
- [Best GUI frameworks for Go - LogRocket](https://blog.logrocket.com/best-gui-frameworks-go/)
