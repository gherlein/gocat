#!/usr/bin/env python3
"""
Plot spectrogram from rf-scanner CSV output.

Usage:
    ./scripts/plot-spectrogram.py spectrum.csv
    ./scripts/plot-spectrogram.py spectrum.csv -o spectrogram.png
"""

import argparse
import sys
import numpy as np

def main():
    parser = argparse.ArgumentParser(description='Plot spectrogram from rf-scanner CSV')
    parser.add_argument('csvfile', help='CSV file from rf-scanner -csv')
    parser.add_argument('-o', '--output', help='Output image file (shows interactive if not specified)')
    parser.add_argument('--cmap', default='viridis', help='Colormap (default: viridis)')
    parser.add_argument('--vmin', type=float, default=-80, help='Min RSSI for color scale (default: -80)')
    parser.add_argument('--vmax', type=float, default=-30, help='Max RSSI for color scale (default: -30)')
    args = parser.parse_args()

    # Import matplotlib here so we can check for its availability
    try:
        import matplotlib.pyplot as plt
    except ImportError:
        print("Error: matplotlib not installed. Install with: pip install matplotlib", file=sys.stderr)
        sys.exit(1)

    # Read CSV
    print(f"Reading {args.csvfile}...")
    with open(args.csvfile, 'r') as f:
        header = f.readline().strip()
        lines = f.readlines()

    # Parse header to get frequencies
    cols = header.split(',')
    freqs = [float(f) for f in cols[1:]]  # Skip timestamp_ms column

    # Parse data rows
    timestamps = []
    rssi_data = []
    for line in lines:
        parts = line.strip().split(',')
        if len(parts) < 2:
            continue
        timestamps.append(int(parts[0]))
        rssi_data.append([float(x) for x in parts[1:]])

    if not rssi_data:
        print("Error: No data rows found in CSV", file=sys.stderr)
        sys.exit(1)

    # Convert to numpy arrays
    rssi_matrix = np.array(rssi_data)
    timestamps = np.array(timestamps)
    freqs = np.array(freqs)

    # Convert timestamps to relative seconds
    t_start = timestamps[0]
    time_sec = (timestamps - t_start) / 1000.0

    print(f"Loaded {len(timestamps)} frames, {len(freqs)} frequency bins")
    print(f"Frequency range: {freqs[0]:.3f} - {freqs[-1]:.3f} MHz")
    print(f"Duration: {time_sec[-1]:.2f} seconds")

    # Create spectrogram plot
    fig, ax = plt.subplots(figsize=(12, 6))

    # Plot as image (time on Y axis, frequency on X axis)
    extent = [freqs[0], freqs[-1], time_sec[-1], time_sec[0]]
    im = ax.imshow(rssi_matrix, aspect='auto', extent=extent,
                   cmap=args.cmap, vmin=args.vmin, vmax=args.vmax,
                   interpolation='nearest')

    ax.set_xlabel('Frequency (MHz)')
    ax.set_ylabel('Time (seconds)')
    ax.set_title(f'RF Spectrogram - {args.csvfile}')

    # Colorbar
    cbar = fig.colorbar(im, ax=ax, label='RSSI (dBm)')

    plt.tight_layout()

    if args.output:
        plt.savefig(args.output, dpi=150)
        print(f"Saved to {args.output}")
    else:
        plt.show()

if __name__ == '__main__':
    main()
