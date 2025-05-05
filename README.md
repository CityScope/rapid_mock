# Rapid Mock

A simple media display system for running videos and images on dual monitors.

only tested in macOS only

## Features

- Display synchronized media content on two separate monitors
- Support for images (jpg, png, jpeg) and videos (mp4, webm, ogg, mov, m4v)
- Simple keyboard controls for navigation
- Automatic video looping
- Full-screen mode

## Installation

### Prerequisites

1. Install Go (Golang)

   - For macOS:
     ```
     brew install go
     ```

   - For Windows:
     Download and install from [golang.org](https://golang.org/dl/)
   - For Linux:
     ```
     sudo apt-get update
     sudo apt-get install golang
     ```

2. Verify installation:
   ```
   go version
   ```

### Setup

1. Clone this repository or download the files

2. Create data directories:
   ```
   mkdir -p data/a data/b
   ```

3. Add your media content:
   - Place videos/images for the top monitor in `data/a/`
   - Place videos/images for the bottom monitor in `data/b/`

## Usage

### Option 1: Run with Go

1. Run the application:
   ```
   go run main.go
   ```

### Option 2: Run with Docker

1. Build the Docker image:
   ```
   docker build -t rapid-mock .
   ```

2. Run the container:
   ```
   docker run -p 8080:8080 -v $(pwd)/data:/app/data rapid-mock
   ```

   This mounts your local `data` directory to the container so it can access your media files.

### Opening the Application

Open browsers on the respective monitors:
- Top monitor browser: `http://localhost:8080/a`
- Bottom monitor browser: `http://localhost:8080/b`

### Controls

- Click anywhere: Advance to next media pair
- `f`: Toggle fullscreen
- Arrow Right: Go to next media pair
- Arrow Left: Go to previous media pair
- `r`: Reload media files from directories
- `0`: Reset to the first media pair

## Notes

- Media files are sorted alphabetically
- Both displays are synchronized to show the same index of content
- Ensure both data/a and data/b have the same number of files for best results
