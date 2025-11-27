# GoCat Spectrum Analyzer GUI Design

## 1. Overview

This document provides a complete design for implementing a real-time spectrum analyzer GUI for GoCat using Fyne and custom canvas rendering. The design mirrors RFCat's spectrum analyzer functionality while leveraging Go's concurrency model and Fyne's cross-platform GUI capabilities.

### 1.1 Goals

- Provide real-time RF spectrum visualization with RSSI/dBm display
- Support interactive frequency and power measurement with cursors
- Maintain 350-frame persistence for maximum signal envelope tracking
- Enable keyboard and mouse-based navigation and control
- Achieve 30-60 FPS rendering performance
- Remain pure Go with no non-Go dependencies
- Provide both library API and standalone application

### 1.2 Non-Goals

- 3D waterfall displays (future enhancement)
- Advanced DSP features beyond RSSI visualization
- Recording/playback (delegated to separate module)
- Network streaming (delegated to separate module)

## 2. System Architecture

### 2.1 High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Fyne Application                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Spectrum Analyzer Window                 │   │
│  │  ┌────────────────────────────────────────────────┐  │   │
│  │  │         Spectrum Display (Canvas)              │  │   │
│  │  │  - Real-time spectrum graph                    │  │   │
│  │  │  - Persistence envelope                        │  │   │
│  │  │  - Grid reticle                                │  │   │
│  │  │  - Measurement cursors                         │  │   │
│  │  └────────────────────────────────────────────────┘  │   │
│  │  ┌────────────────────────────────────────────────┐  │   │
│  │  │         Control Panel                          │  │   │
│  │  │  - Frequency controls                          │  │   │
│  │  │  - Span/resolution controls                    │  │   │
│  │  │  - Display options                             │  │   │
│  │  │  - Status display                              │  │   │
│  │  └────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  Spectrum Data Pipeline                      │
│                                                               │
│  RF Device → Data Acquisition → Calibration → Persistence   │
│              Goroutine          Processing     Buffer        │
│                                                               │
│                              ↓                                │
│                      Frame Channel                            │
│                              ↓                                │
│                       GUI Update Loop                         │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Component Breakdown

#### 2.2.1 SpectrumAnalyzer

Main coordinator that owns the window, manages data flow, and coordinates between components.

#### 2.2.2 SpectrumCanvas

Custom Fyne canvas widget that handles rendering of spectrum data, grid, and cursors.

#### 2.2.3 PersistenceBuffer

Circular buffer maintaining historical frames for maximum envelope tracking.

#### 2.2.4 SpectrumDataSource

Interface abstracting the RF device for testability and flexibility.

#### 2.2.5 CoordinateMapper

Handles transformations between frequency/power domain and pixel coordinates.

#### 2.2.6 ControlPanel

UI controls for configuring spectrum analyzer parameters.

## 3. Data Structures

### 3.1 Core Types

```go
package spectrumanalyzer

import (
    "image"
    "sync"
    "time"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
)

// SpectrumFrame represents a single sweep of the spectrum
type SpectrumFrame struct {
    Frequencies []float64  // Hz values for each bin
    RSSI        []float64  // RSSI values in dBm
    Timestamp   time.Time  // When this frame was captured
    CenterFreq  float64    // Center frequency in Hz
    Span        float64    // Frequency span in Hz
    Resolution  float64    // Frequency resolution (bin width) in Hz
}

// SpectrumConfig holds configuration for spectrum analysis
type SpectrumConfig struct {
    CenterFreq    float64  // Center frequency in Hz
    Span          float64  // Frequency span in Hz
    BinCount      int      // Number of frequency bins (max 255)
    UpdateRate    int      // Target updates per second
    Persistence   int      // Number of frames to keep for max hold (default 350)
    
    // Display configuration
    MinDBm        float64  // Minimum dBm for Y axis
    MaxDBm        float64  // Maximum dBm for Y axis
    ShowGrid      bool     // Show grid reticle
    ShowPersistence bool   // Show max envelope
    ShowCursors   bool     // Show measurement cursors
}

// CursorPosition represents a measurement cursor
type CursorPosition struct {
    Frequency float64  // Hz
    Power     float64  // dBm
    Active    bool     // Whether cursor is visible
}

// SpectrumAnalyzer is the main spectrum analyzer component
type SpectrumAnalyzer struct {
    config        SpectrumConfig
    dataSource    SpectrumDataSource
    
    // Data management
    persistence   *PersistenceBuffer
    currentFrame  *SpectrumFrame
    frameMutex    sync.RWMutex
    
    // UI components
    window        fyne.Window
    canvas        *SpectrumCanvas
    controls      *ControlPanel
    
    // Cursors
    leftCursor    CursorPosition
    rightCursor   CursorPosition
    cursorMutex   sync.RWMutex
    
    // Coordinate mapping
    mapper        *CoordinateMapper
    
    // Lifecycle
    running       bool
    stopChan      chan struct{}
    frameChan     chan *SpectrumFrame
}

// PersistenceBuffer maintains historical spectrum frames
type PersistenceBuffer struct {
    frames    []*SpectrumFrame
    maxFrames int
    index     int
    mutex     sync.RWMutex
}

// SpectrumCanvas is the custom Fyne widget for rendering
type SpectrumCanvas struct {
    widget.BaseWidget
    
    analyzer   *SpectrumAnalyzer
    raster     *canvas.Raster
    
    // Rendering state
    width      int
    height     int
    imageCache *image.RGBA
    dirty      bool
    mutex      sync.RWMutex
}

// CoordinateMapper handles coordinate transformations
type CoordinateMapper struct {
    config     *SpectrumConfig
    viewWidth  int
    viewHeight int
    mutex      sync.RWMutex
}

// ControlPanel manages UI controls
type ControlPanel struct {
    analyzer      *SpectrumAnalyzer
    
    // Control widgets
    centerFreqEntry    *widget.Entry
    spanSlider         *widget.Slider
    resolutionSelect   *widget.Select
    startButton        *widget.Button
    stopButton         *widget.Button
    
    // Status widgets
    statusLabel        *widget.Label
    fpsLabel           *widget.Label
    cursorInfoLabel    *widget.Label
}
```

### 3.2 Interfaces

```go
// SpectrumDataSource provides spectrum data from RF device
type SpectrumDataSource interface {
    // Configure sets up the spectrum analyzer parameters
    Configure(config SpectrumConfig) error
    
    // Start begins spectrum data acquisition
    Start() error
    
    // Stop halts data acquisition
    Stop() error
    
    // Stream returns a channel that emits spectrum frames
    Stream() <-chan *SpectrumFrame
    
    // IsRunning returns true if actively scanning
    IsRunning() bool
}

// Renderer defines the rendering interface
type Renderer interface {
    // Render generates the spectrum image
    Render(width, height int) *image.RGBA
}
```

## 4. Detailed Component Design

### 4.1 SpectrumAnalyzer

The main coordinator component.

```go
// NewSpectrumAnalyzer creates a new spectrum analyzer
func NewSpectrumAnalyzer(dataSource SpectrumDataSource, config SpectrumConfig) *SpectrumAnalyzer {
    sa := &SpectrumAnalyzer{
        config:       config,
        dataSource:   dataSource,
        persistence:  NewPersistenceBuffer(config.Persistence),
        mapper:       NewCoordinateMapper(&config),
        stopChan:     make(chan struct{}),
        frameChan:    make(chan *SpectrumFrame, 10),
    }
    
    sa.canvas = NewSpectrumCanvas(sa)
    sa.controls = NewControlPanel(sa)
    
    return sa
}

// Show displays the spectrum analyzer window
func (sa *SpectrumAnalyzer) Show() {
    app := app.New()
    sa.window = app.NewWindow("GoCat Spectrum Analyzer")
    
    // Create layout
    content := container.NewBorder(
        nil,              // top
        sa.controls.Build(), // bottom
        nil,              // left
        nil,              // right
        sa.canvas,        // center
    )
    
    sa.window.SetContent(content)
    sa.window.Resize(fyne.NewSize(1024, 768))
    
    // Setup keyboard shortcuts
    sa.setupKeyboardHandlers()
    
    sa.window.ShowAndRun()
}

// Start begins spectrum analysis
func (sa *SpectrumAnalyzer) Start() error {
    if sa.running {
        return nil
    }
    
    if err := sa.dataSource.Configure(sa.config); err != nil {
        return err
    }
    
    if err := sa.dataSource.Start(); err != nil {
        return err
    }
    
    sa.running = true
    go sa.dataAcquisitionLoop()
    go sa.renderLoop()
    
    return nil
}

// Stop halts spectrum analysis
func (sa *SpectrumAnalyzer) Stop() {
    if !sa.running {
        return
    }
    
    sa.running = false
    close(sa.stopChan)
    sa.dataSource.Stop()
}

// dataAcquisitionLoop receives frames from data source
func (sa *SpectrumAnalyzer) dataAcquisitionLoop() {
    stream := sa.dataSource.Stream()
    
    for {
        select {
        case frame := <-stream:
            sa.frameMutex.Lock()
            sa.currentFrame = frame
            sa.persistence.Add(frame)
            sa.frameMutex.Unlock()
            
            // Trigger render
            sa.canvas.Refresh()
            
        case <-sa.stopChan:
            return
        }
    }
}

// renderLoop handles periodic UI updates
func (sa *SpectrumAnalyzer) renderLoop() {
    ticker := time.NewTicker(time.Second / time.Duration(sa.config.UpdateRate))
    defer ticker.Stop()
    
    frameCount := 0
    fpsStart := time.Now()
    
    for {
        select {
        case <-ticker.C:
            frameCount++
            
            // Update FPS counter every second
            if time.Since(fpsStart) >= time.Second {
                fps := float64(frameCount) / time.Since(fpsStart).Seconds()
                sa.controls.UpdateFPS(fps)
                frameCount = 0
                fpsStart = time.Now()
            }
            
        case <-sa.stopChan:
            return
        }
    }
}

// SetCenterFrequency updates the center frequency
func (sa *SpectrumAnalyzer) SetCenterFrequency(freq float64) error {
    sa.config.CenterFreq = freq
    return sa.reconfigure()
}

// SetSpan updates the frequency span
func (sa *SpectrumAnalyzer) SetSpan(span float64) error {
    sa.config.Span = span
    return sa.reconfigure()
}

// reconfigure applies configuration changes
func (sa *SpectrumAnalyzer) reconfigure() error {
    if sa.running {
        return sa.dataSource.Configure(sa.config)
    }
    return nil
}

// setupKeyboardHandlers configures keyboard shortcuts
func (sa *SpectrumAnalyzer) setupKeyboardHandlers() {
    // Arrow keys for navigation
    // H for help
    // C to clear persistence
    // Space to toggle pause
    // Etc.
}
```

### 4.2 SpectrumCanvas

Custom canvas widget for rendering.

```go
// NewSpectrumCanvas creates a new spectrum canvas
func NewSpectrumCanvas(analyzer *SpectrumAnalyzer) *SpectrumCanvas {
    sc := &SpectrumCanvas{
        analyzer: analyzer,
    }
    
    sc.raster = canvas.NewRasterWithPixels(sc.generatePixels)
    sc.ExtendBaseWidget(sc)
    
    return sc
}

// CreateRenderer implements fyne.Widget
func (sc *SpectrumCanvas) CreateRenderer() fyne.WidgetRenderer {
    return &spectrumCanvasRenderer{
        canvas: sc,
        raster: sc.raster,
    }
}

// generatePixels is called by Fyne for each pixel
func (sc *SpectrumCanvas) generatePixels(x, y, w, h int) color.Color {
    sc.mutex.RLock()
    defer sc.mutex.RUnlock()
    
    // Check if we have cached image
    if sc.imageCache != nil && sc.width == w && sc.height == h && !sc.dirty {
        return sc.imageCache.At(x, y)
    }
    
    // Need to regenerate
    if sc.imageCache == nil || sc.width != w || sc.height != h {
        sc.mutex.RUnlock()
        sc.mutex.Lock()
        sc.regenerateImage(w, h)
        sc.width = w
        sc.height = h
        sc.dirty = false
        sc.mutex.Unlock()
        sc.mutex.RLock()
    }
    
    return sc.imageCache.At(x, y)
}

// regenerateImage creates the full spectrum image
func (sc *SpectrumCanvas) regenerateImage(width, height int) {
    img := image.NewRGBA(image.Rect(0, 0, width, height))
    
    // Draw background
    sc.drawBackground(img)
    
    // Draw grid reticle
    if sc.analyzer.config.ShowGrid {
        sc.drawGrid(img)
    }
    
    // Draw persistence envelope
    if sc.analyzer.config.ShowPersistence {
        sc.drawPersistence(img)
    }
    
    // Draw current spectrum
    sc.drawSpectrum(img)
    
    // Draw cursors
    if sc.analyzer.config.ShowCursors {
        sc.drawCursors(img)
    }
    
    sc.imageCache = img
}

// drawBackground fills background
func (sc *SpectrumCanvas) drawBackground(img *image.RGBA) {
    bounds := img.Bounds()
    bgColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
    
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            img.Set(x, y, bgColor)
        }
    }
}

// drawGrid renders the frequency/power grid
func (sc *SpectrumCanvas) drawGrid(img *image.RGBA) {
    bounds := img.Bounds()
    width := bounds.Dx()
    height := bounds.Dy()
    gridColor := color.RGBA{R: 50, G: 50, B: 50, A: 255}
    
    // Update mapper dimensions
    sc.analyzer.mapper.SetDimensions(width, height)
    
    // Vertical lines (frequency)
    freqStep := sc.calculateFrequencyStep()
    startFreq := sc.analyzer.config.CenterFreq - sc.analyzer.config.Span/2
    endFreq := sc.analyzer.config.CenterFreq + sc.analyzer.config.Span/2
    
    for freq := startFreq; freq <= endFreq; freq += freqStep {
        x := sc.analyzer.mapper.FrequencyToX(freq)
        if x >= 0 && x < width {
            sc.drawVerticalLine(img, x, gridColor)
        }
    }
    
    // Horizontal lines (power)
    powerStep := sc.calculatePowerStep()
    for dbm := sc.analyzer.config.MinDBm; dbm <= sc.analyzer.config.MaxDBm; dbm += powerStep {
        y := sc.analyzer.mapper.PowerToY(dbm)
        if y >= 0 && y < height {
            sc.drawHorizontalLine(img, y, gridColor)
        }
    }
}

// drawSpectrum renders the current spectrum line
func (sc *SpectrumCanvas) drawSpectrum(img *image.RGBA) {
    sc.analyzer.frameMutex.RLock()
    frame := sc.analyzer.currentFrame
    sc.analyzer.frameMutex.RUnlock()
    
    if frame == nil || len(frame.Frequencies) == 0 {
        return
    }
    
    width := img.Bounds().Dx()
    height := img.Bounds().Dy()
    sc.analyzer.mapper.SetDimensions(width, height)
    
    spectrumColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
    
    // Draw line connecting points
    for i := 0; i < len(frame.Frequencies)-1; i++ {
        x1 := sc.analyzer.mapper.FrequencyToX(frame.Frequencies[i])
        y1 := sc.analyzer.mapper.PowerToY(frame.RSSI[i])
        x2 := sc.analyzer.mapper.FrequencyToX(frame.Frequencies[i+1])
        y2 := sc.analyzer.mapper.PowerToY(frame.RSSI[i+1])
        
        sc.drawLine(img, x1, y1, x2, y2, spectrumColor)
    }
}

// drawPersistence renders the max envelope
func (sc *SpectrumCanvas) drawPersistence(img *image.RGBA) {
    maxEnvelope := sc.analyzer.persistence.MaxEnvelope()
    if maxEnvelope == nil {
        return
    }
    
    width := img.Bounds().Dx()
    height := img.Bounds().Dy()
    sc.analyzer.mapper.SetDimensions(width, height)
    
    persistColor := color.RGBA{R: 0, G: 255, B: 0, A: 128}
    
    // Get frequency array from current or first frame
    sc.analyzer.frameMutex.RLock()
    frame := sc.analyzer.currentFrame
    sc.analyzer.frameMutex.RUnlock()
    
    if frame == nil {
        return
    }
    
    for i := 0; i < len(maxEnvelope)-1; i++ {
        x1 := sc.analyzer.mapper.FrequencyToX(frame.Frequencies[i])
        y1 := sc.analyzer.mapper.PowerToY(maxEnvelope[i])
        x2 := sc.analyzer.mapper.FrequencyToX(frame.Frequencies[i+1])
        y2 := sc.analyzer.mapper.PowerToY(maxEnvelope[i+1])
        
        sc.drawLine(img, x1, y1, x2, y2, persistColor)
    }
}

// drawCursors renders measurement cursors
func (sc *SpectrumCanvas) drawCursors(img *image.RGBA) {
    width := img.Bounds().Dx()
    height := img.Bounds().Dy()
    sc.analyzer.mapper.SetDimensions(width, height)
    
    sc.analyzer.cursorMutex.RLock()
    defer sc.analyzer.cursorMutex.RUnlock()
    
    leftColor := color.RGBA{R: 255, G: 255, B: 0, A: 255}
    rightColor := color.RGBA{R: 0, G: 255, B: 255, A: 255}
    
    if sc.analyzer.leftCursor.Active {
        x := sc.analyzer.mapper.FrequencyToX(sc.analyzer.leftCursor.Frequency)
        y := sc.analyzer.mapper.PowerToY(sc.analyzer.leftCursor.Power)
        sc.drawCrosshair(img, x, y, leftColor)
    }
    
    if sc.analyzer.rightCursor.Active {
        x := sc.analyzer.mapper.FrequencyToX(sc.analyzer.rightCursor.Frequency)
        y := sc.analyzer.mapper.PowerToY(sc.analyzer.rightCursor.Power)
        sc.drawCrosshair(img, x, y, rightColor)
    }
}

// Helper drawing functions
func (sc *SpectrumCanvas) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
    // Bresenham's line algorithm
    dx := abs(x2 - x1)
    dy := abs(y2 - y1)
    sx := 1
    if x1 >= x2 {
        sx = -1
    }
    sy := 1
    if y1 >= y2 {
        sy = -1
    }
    err := dx - dy
    
    for {
        if x1 >= 0 && x1 < img.Bounds().Dx() && y1 >= 0 && y1 < img.Bounds().Dy() {
            img.Set(x1, y1, c)
        }
        
        if x1 == x2 && y1 == y2 {
            break
        }
        
        e2 := 2 * err
        if e2 > -dy {
            err -= dy
            x1 += sx
        }
        if e2 < dx {
            err += dx
            y1 += sy
        }
    }
}

func (sc *SpectrumCanvas) drawVerticalLine(img *image.RGBA, x int, c color.Color) {
    for y := 0; y < img.Bounds().Dy(); y++ {
        img.Set(x, y, c)
    }
}

func (sc *SpectrumCanvas) drawHorizontalLine(img *image.RGBA, y int, c color.Color) {
    for x := 0; x < img.Bounds().Dx(); x++ {
        img.Set(x, y, c)
    }
}

func (sc *SpectrumCanvas) drawCrosshair(img *image.RGBA, x, y int, c color.Color) {
    // Draw horizontal and vertical lines
    sc.drawHorizontalLine(img, y, c)
    sc.drawVerticalLine(img, x, c)
}

// Mouse event handling
func (sc *SpectrumCanvas) Tapped(event *fyne.PointEvent) {
    // Left click - set left cursor
    x := int(event.Position.X)
    y := int(event.Position.Y)
    
    freq := sc.analyzer.mapper.XToFrequency(x)
    power := sc.analyzer.mapper.YToPower(y)
    
    sc.analyzer.cursorMutex.Lock()
    sc.analyzer.leftCursor = CursorPosition{
        Frequency: freq,
        Power:     power,
        Active:    true,
    }
    sc.analyzer.cursorMutex.Unlock()
    
    sc.dirty = true
    sc.Refresh()
}

func (sc *SpectrumCanvas) TappedSecondary(event *fyne.PointEvent) {
    // Right click - set right cursor
    x := int(event.Position.X)
    y := int(event.Position.Y)
    
    freq := sc.analyzer.mapper.XToFrequency(x)
    power := sc.analyzer.mapper.YToPower(y)
    
    sc.analyzer.cursorMutex.Lock()
    sc.analyzer.rightCursor = CursorPosition{
        Frequency: freq,
        Power:     power,
        Active:    true,
    }
    sc.analyzer.cursorMutex.Unlock()
    
    sc.dirty = true
    sc.Refresh()
}
```

### 4.3 PersistenceBuffer

Manages historical frames for max envelope.

```go
// NewPersistenceBuffer creates a new persistence buffer
func NewPersistenceBuffer(maxFrames int) *PersistenceBuffer {
    return &PersistenceBuffer{
        frames:    make([]*SpectrumFrame, 0, maxFrames),
        maxFrames: maxFrames,
        index:     0,
    }
}

// Add adds a new frame to the buffer
func (pb *PersistenceBuffer) Add(frame *SpectrumFrame) {
    pb.mutex.Lock()
    defer pb.mutex.Unlock()
    
    if len(pb.frames) < pb.maxFrames {
        pb.frames = append(pb.frames, frame)
    } else {
        pb.frames[pb.index] = frame
        pb.index = (pb.index + 1) % pb.maxFrames
    }
}

// MaxEnvelope computes the maximum RSSI across all frames
func (pb *PersistenceBuffer) MaxEnvelope() []float64 {
    pb.mutex.RLock()
    defer pb.mutex.RUnlock()
    
    if len(pb.frames) == 0 {
        return nil
    }
    
    // Get bin count from first frame
    binCount := len(pb.frames[0].RSSI)
    maxValues := make([]float64, binCount)
    
    // Initialize with minimum values
    for i := range maxValues {
        maxValues[i] = -200.0 // Very low dBm
    }
    
    // Find maximum at each bin
    for _, frame := range pb.frames {
        if frame == nil {
            continue
        }
        for i, rssi := range frame.RSSI {
            if rssi > maxValues[i] {
                maxValues[i] = rssi
            }
        }
    }
    
    return maxValues
}

// Clear removes all frames
func (pb *PersistenceBuffer) Clear() {
    pb.mutex.Lock()
    defer pb.mutex.Unlock()
    
    pb.frames = make([]*SpectrumFrame, 0, pb.maxFrames)
    pb.index = 0
}

// Count returns the number of frames stored
func (pb *PersistenceBuffer) Count() int {
    pb.mutex.RLock()
    defer pb.mutex.RUnlock()
    
    return len(pb.frames)
}
```

### 4.4 CoordinateMapper

Handles coordinate transformations.

```go
// NewCoordinateMapper creates a new coordinate mapper
func NewCoordinateMapper(config *SpectrumConfig) *CoordinateMapper {
    return &CoordinateMapper{
        config: config,
    }
}

// SetDimensions updates the view dimensions
func (cm *CoordinateMapper) SetDimensions(width, height int) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    cm.viewWidth = width
    cm.viewHeight = height
}

// FrequencyToX converts frequency to X pixel coordinate
func (cm *CoordinateMapper) FrequencyToX(freq float64) int {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    minFreq := cm.config.CenterFreq - cm.config.Span/2
    maxFreq := cm.config.CenterFreq + cm.config.Span/2
    
    if maxFreq <= minFreq {
        return 0
    }
    
    normalized := (freq - minFreq) / (maxFreq - minFreq)
    return int(normalized * float64(cm.viewWidth))
}

// PowerToY converts power (dBm) to Y pixel coordinate
func (cm *CoordinateMapper) PowerToY(power float64) int {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    if cm.config.MaxDBm <= cm.config.MinDBm {
        return 0
    }
    
    // Invert Y axis (higher power = lower Y pixel)
    normalized := (power - cm.config.MinDBm) / (cm.config.MaxDBm - cm.config.MinDBm)
    return cm.viewHeight - int(normalized*float64(cm.viewHeight))
}

// XToFrequency converts X pixel coordinate to frequency
func (cm *CoordinateMapper) XToFrequency(x int) float64 {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    if cm.viewWidth == 0 {
        return cm.config.CenterFreq
    }
    
    minFreq := cm.config.CenterFreq - cm.config.Span/2
    maxFreq := cm.config.CenterFreq + cm.config.Span/2
    
    normalized := float64(x) / float64(cm.viewWidth)
    return minFreq + normalized*(maxFreq-minFreq)
}

// YToPower converts Y pixel coordinate to power (dBm)
func (cm *CoordinateMapper) YToPower(y int) float64 {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    if cm.viewHeight == 0 {
        return cm.config.MinDBm
    }
    
    // Invert Y axis
    normalized := float64(cm.viewHeight-y) / float64(cm.viewHeight)
    return cm.config.MinDBm + normalized*(cm.config.MaxDBm-cm.config.MinDBm)
}
```

### 4.5 ControlPanel

UI controls for configuration.

```go
// NewControlPanel creates a new control panel
func NewControlPanel(analyzer *SpectrumAnalyzer) *ControlPanel {
    return &ControlPanel{
        analyzer: analyzer,
    }
}

// Build constructs the control panel UI
func (cp *ControlPanel) Build() fyne.CanvasObject {
    // Center frequency control
    cp.centerFreqEntry = widget.NewEntry()
    cp.centerFreqEntry.SetText(formatFrequency(cp.analyzer.config.CenterFreq))
    cp.centerFreqEntry.OnSubmitted = func(text string) {
        freq, err := parseFrequency(text)
        if err == nil {
            cp.analyzer.SetCenterFrequency(freq)
        }
    }
    
    // Span control
    cp.spanSlider = widget.NewSlider(100e3, 100e6)
    cp.spanSlider.Value = cp.analyzer.config.Span
    cp.spanSlider.OnChanged = func(value float64) {
        cp.analyzer.SetSpan(value)
    }
    
    // Start/Stop buttons
    cp.startButton = widget.NewButton("Start", func() {
        cp.analyzer.Start()
        cp.startButton.Disable()
        cp.stopButton.Enable()
    })
    
    cp.stopButton = widget.NewButton("Stop", func() {
        cp.analyzer.Stop()
        cp.stopButton.Disable()
        cp.startButton.Enable()
    })
    cp.stopButton.Disable()
    
    // Status labels
    cp.statusLabel = widget.NewLabel("Ready")
    cp.fpsLabel = widget.NewLabel("FPS: 0")
    cp.cursorInfoLabel = widget.NewLabel("Cursors: None")
    
    // Layout
    freqBox := container.NewHBox(
        widget.NewLabel("Center Freq:"),
        cp.centerFreqEntry,
        widget.NewLabel("Span:"),
        cp.spanSlider,
    )
    
    buttonBox := container.NewHBox(
        cp.startButton,
        cp.stopButton,
    )
    
    statusBox := container.NewHBox(
        cp.statusLabel,
        cp.fpsLabel,
        cp.cursorInfoLabel,
    )
    
    return container.NewVBox(
        freqBox,
        buttonBox,
        statusBox,
    )
}

// UpdateFPS updates the FPS display
func (cp *ControlPanel) UpdateFPS(fps float64) {
    cp.fpsLabel.SetText(fmt.Sprintf("FPS: %.1f", fps))
}

// UpdateCursorInfo updates cursor information display
func (cp *ControlPanel) UpdateCursorInfo(left, right CursorPosition) {
    if !left.Active && !right.Active {
        cp.cursorInfoLabel.SetText("Cursors: None")
        return
    }
    
    var text string
    if left.Active && right.Active {
        deltaFreq := right.Frequency - left.Frequency
        deltaPower := right.Power - left.Power
        text = fmt.Sprintf("ΔF: %.3f MHz, ΔP: %.1f dBm", deltaFreq/1e6, deltaPower)
    } else if left.Active {
        text = fmt.Sprintf("L: %.3f MHz, %.1f dBm", left.Frequency/1e6, left.Power)
    } else {
        text = fmt.Sprintf("R: %.3f MHz, %.1f dBm", right.Frequency/1e6, right.Power)
    }
    
    cp.cursorInfoLabel.SetText(text)
}
```

## 5. Data Flow

### 5.1 Initialization Flow

```
1. User creates SpectrumAnalyzer with config and data source
2. SpectrumAnalyzer creates:
   - PersistenceBuffer
   - CoordinateMapper
   - SpectrumCanvas
   - ControlPanel
3. User calls Show()
4. Fyne window created with layout
5. Event handlers registered
```

### 5.2 Runtime Data Flow

```
RF Device → SpectrumDataSource.Stream()
              ↓
          Frame Channel
              ↓
    dataAcquisitionLoop()
              ↓
    Updates currentFrame and persistence buffer
              ↓
    Triggers canvas.Refresh()
              ↓
    Fyne calls generatePixels()
              ↓
    regenerateImage() creates new image
              ↓
    Display updated
```

### 5.3 User Interaction Flow

```
Mouse Click → Tapped() or TappedSecondary()
              ↓
         Convert pixel to freq/power
              ↓
         Update cursor position
              ↓
         Mark canvas dirty
              ↓
         Refresh canvas
              ↓
         Update cursor info label
```

## 6. Concurrency Model

### 6.1 Goroutines

1. **Data Acquisition Goroutine**: Reads from data source stream
2. **Render Loop Goroutine**: Periodic FPS updates and statistics
3. **Fyne Main Loop**: Handles UI events (managed by Fyne)

### 6.2 Synchronization

- `frameMutex`: Protects currentFrame access
- `cursorMutex`: Protects cursor positions
- `PersistenceBuffer.mutex`: Protects frame buffer
- `CoordinateMapper.mutex`: Protects coordinate transformation state
- `SpectrumCanvas.mutex`: Protects rendering state

### 6.3 Channels

- `stopChan`: Signal to stop all goroutines
- `frameChan`: Optional buffered channel for frame delivery
- Data source stream channel

## 7. Performance Optimization

### 7.1 Rendering Optimizations

1. **Image Caching**: Cache rendered image, only regenerate when dirty
2. **Dirty Flag**: Track when data changes require re-render
3. **Partial Updates**: Future enhancement for incremental updates
4. **Batch Drawing**: Use efficient line drawing algorithms

### 7.2 Data Management

1. **Circular Buffer**: Fixed-size persistence buffer avoids allocations
2. **Reuse Frames**: Pool frame objects to reduce GC pressure
3. **Efficient Slices**: Pre-allocate slices for known sizes

### 7.3 Concurrency

1. **Non-blocking Reads**: Use RWMutex for read-heavy operations
2. **Buffered Channels**: Prevent blocking on frame delivery
3. **Goroutine Pooling**: Future enhancement if needed

## 8. User Interaction Design

### 8.1 Mouse Controls

- **Left Click**: Set left measurement cursor
- **Right Click**: Set right measurement cursor
- **Middle Click**: Clear cursors / toggle overlay
- **Scroll Wheel**: Zoom in/out on frequency span
- **Drag**: Pan frequency range (future)

### 8.2 Keyboard Shortcuts

- **Arrow Up/Down**: Adjust center frequency
- **Arrow Left/Right**: Adjust span
- **+/-**: Zoom in/out
- **Space**: Pause/resume acquisition
- **C**: Clear persistence buffer
- **H**: Show help overlay
- **ESC**: Close window
- **S**: Save screenshot

### 8.3 Touch Controls (future)

- **Pinch**: Zoom
- **Two-finger drag**: Pan

## 9. Configuration and Settings

### 9.1 Default Configuration

```go
var DefaultConfig = SpectrumConfig{
    CenterFreq:      433.92e6,    // 433.92 MHz
    Span:            10e6,         // 10 MHz
    BinCount:        104,          // Match RFCat default
    UpdateRate:      30,           // 30 FPS
    Persistence:     350,          // 350 frames
    MinDBm:          -110,
    MaxDBm:          -30,
    ShowGrid:        true,
    ShowPersistence: true,
    ShowCursors:     true,
}
```

### 9.2 Persistence

Save/load configuration from JSON files.

```go
// SaveConfig saves configuration to file
func (sa *SpectrumAnalyzer) SaveConfig(filename string) error

// LoadConfig loads configuration from file
func (sa *SpectrumAnalyzer) LoadConfig(filename string) error
```

## 10. Testing Strategy

### 10.1 Unit Tests

- CoordinateMapper transformations
- PersistenceBuffer operations
- Frame calculations
- Configuration validation

### 10.2 Integration Tests

- Mock SpectrumDataSource for deterministic testing
- Verify data flow through pipeline
- Test cursor positioning
- Test configuration changes

### 10.3 Visual Tests

- Screenshot comparison for rendering
- Manual testing of UI interactions
- Performance profiling

### 10.4 Testable Mock Data Source

```go
type MockDataSource struct {
    config  SpectrumConfig
    running bool
    stream  chan *SpectrumFrame
}

func NewMockDataSource() *MockDataSource {
    return &MockDataSource{
        stream: make(chan *SpectrumFrame, 10),
    }
}

func (m *MockDataSource) Configure(config SpectrumConfig) error {
    m.config = config
    return nil
}

func (m *MockDataSource) Start() error {
    m.running = true
    go m.generateTestData()
    return nil
}

func (m *MockDataSource) generateTestData() {
    // Generate synthetic spectrum data
    // Sine wave, noise, or recorded data
}
```

## 11. Error Handling

### 11.1 Device Errors

- Connection loss: Display error, attempt reconnection
- Configuration errors: Validate before applying
- Timeout errors: Log and continue

### 11.2 Rendering Errors

- Invalid dimensions: Use safe defaults
- Out of range values: Clamp to valid range
- Memory allocation failures: Reduce buffer sizes

### 11.3 User Input Errors

- Invalid frequency: Show error message, revert
- Invalid configuration: Highlight problematic field
- File I/O errors: Display error dialog

## 12. Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- Implement data structures
- Create CoordinateMapper
- Build PersistenceBuffer
- Unit tests

### Phase 2: Basic Rendering (Week 2)
- Implement SpectrumCanvas
- Basic line drawing
- Grid rendering
- Background/foreground colors

### Phase 3: Data Integration (Week 3)
- Implement SpectrumDataSource interface
- Create mock data source
- Data acquisition loop
- Frame processing

### Phase 4: Interactive Features (Week 4)
- Mouse cursor support
- Keyboard navigation
- Cursor position display
- Persistence visualization

### Phase 5: Control Panel (Week 5)
- Build UI controls
- Configuration binding
- Status display
- FPS counter

### Phase 6: Polish and Optimization (Week 6)
- Performance profiling
- Rendering optimization
- Error handling
- Documentation

### Phase 7: Advanced Features (Future)
- Save/load configurations
- Screenshot export
- Help system
- Themes

## 13. File Structure

```
gocat/
├── spectrumanalyzer/
│   ├── analyzer.go           # Main SpectrumAnalyzer
│   ├── canvas.go             # SpectrumCanvas widget
│   ├── controls.go           # ControlPanel
│   ├── mapper.go             # CoordinateMapper
│   ├── persistence.go        # PersistenceBuffer
│   ├── types.go              # Data structures
│   ├── datasource.go         # SpectrumDataSource interface
│   ├── mock_datasource.go    # Mock for testing
│   ├── analyzer_test.go
│   ├── mapper_test.go
│   └── persistence_test.go
├── cmd/
│   └── spectrumanalyzer/
│       └── main.go           # Standalone application
└── examples/
    └── spectrum/
        └── example.go        # Usage example
```

## 14. API Examples

### 14.1 Basic Usage

```go
package main

import (
    "github.com/yourusername/gocat/spectrumanalyzer"
    "github.com/yourusername/gocat/device"
)

func main() {
    // Create RF device connection
    rfDevice, err := device.Open("/dev/ttyUSB0")
    if err != nil {
        panic(err)
    }
    defer rfDevice.Close()
    
    // Create data source
    dataSource := spectrumanalyzer.NewRFDataSource(rfDevice)
    
    // Create spectrum analyzer with default config
    config := spectrumanalyzer.DefaultConfig
    config.CenterFreq = 915e6  // 915 MHz
    config.Span = 20e6         // 20 MHz span
    
    analyzer := spectrumanalyzer.NewSpectrumAnalyzer(dataSource, config)
    
    // Show GUI (blocks until window closes)
    analyzer.Show()
}
```

### 14.2 Programmatic Control

```go
// Create analyzer
analyzer := spectrumanalyzer.NewSpectrumAnalyzer(dataSource, config)

// Start acquisition
analyzer.Start()

// Change parameters while running
analyzer.SetCenterFrequency(433.92e6)
analyzer.SetSpan(10e6)

// Access current data
frame := analyzer.GetCurrentFrame()
maxEnvelope := analyzer.GetPersistenceEnvelope()

// Stop acquisition
analyzer.Stop()
```

### 14.3 Headless Mode

```go
// Use without GUI
analyzer := spectrumanalyzer.NewSpectrumAnalyzer(dataSource, config)
analyzer.Start()

// Subscribe to frames
frameChan := analyzer.SubscribeFrames()
for frame := range frameChan {
    // Process frame data
    analyzeSpectrum(frame)
}
```

## 15. Dependencies

### 15.1 Required

```go
// go.mod
module github.com/yourusername/gocat

go 1.21

require (
    fyne.io/fyne/v2 v2.4.0
)
```

### 15.2 Optional

- None (pure Go implementation)

## 16. Future Enhancements

### 16.1 Visualization

- Waterfall display (time-frequency)
- 3D spectrum view
- Color gradient mapping
- Multiple simultaneous views

### 16.2 Analysis

- Peak detection and labeling
- Signal classification
- Bandwidth measurement
- Modulation identification hints

### 16.3 Recording

- Save spectrum data to file
- Playback recorded data
- Export to CSV/JSON
- Video recording of display

### 16.4 Advanced UI

- Configurable color schemes
- Multiple measurement cursors
- Markers for known frequencies
- Annotation system

### 16.5 Performance

- GPU-accelerated rendering
- Shader-based effects
- Parallel processing
- Adaptive frame rate

## 17. Summary

This design provides a complete blueprint for implementing a production-quality spectrum analyzer GUI for GoCat using Fyne. The architecture balances:

- **Performance**: 30-60 FPS real-time rendering
- **Maintainability**: Clean separation of concerns
- **Testability**: Mock interfaces and unit tests
- **Usability**: Intuitive mouse/keyboard controls
- **Extensibility**: Easy to add features

The design mirrors RFCat's proven approach while leveraging Go's strengths in concurrency and Fyne's cross-platform capabilities. All components are pure Go with no non-Go dependencies, meeting project requirements.
