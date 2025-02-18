package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var (
	mu           sync.Mutex
	currentIndex int
	filesA       []string
	filesB       []string
)

// SSE client management.
var (
	sseMu      sync.Mutex
	sseClients = make(map[chan string]bool)
)

// Templates.
var (
	fullPageTmpl = template.Must(template.New("fullPage").Parse(fullPageHTML))
	contentTmpl  = template.Must(template.New("content").Parse(contentTemplateHTML))
)

// Full page template includes htmx, SSE connection, a click overlay, and keydown listener for full screen toggling.
const fullPageHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>{{.Title}}</title>
	<script src="https://unpkg.com/htmx.org@1.9.12"></script>
	<style>
		html, body { margin: 0; padding: 0; height: 100%; background: black; }
		/* The overlay captures clicks to advance the slide */
		#click-overlay {
			position: fixed;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			z-index: 999;
			cursor: pointer;
		}
	</style>
</head>
<body hx-sse="connect:/sse">
	<!-- This container is updated via htmx on load and when a "mediaChanged" SSE event is received -->
	<div id="media-container" hx-get="{{.ContentURL}}" hx-trigger="load, sse:mediaChanged" hx-swap="innerHTML">
		Loading...
	</div>
	<!-- A transparent overlay to capture clicks -->
	<div id="click-overlay" hx-post="/advance" hx-trigger="click" hx-swap="none"></div>
	<script>
		// Toggle full screen mode when 'f' (or 'F') is pressed.
		document.addEventListener("keydown", function(e) {
			if (e.key === "f" || e.key === "F") {
				if (!document.fullscreenElement) {
					document.documentElement.requestFullscreen().catch(err => {
						console.error("Error enabling full-screen mode:", err);
					});
				} else {
					document.exitFullscreen();
				}
			}
		});
	</script>
</body>
</html>
`

// Partial template to render the media element.
const contentTemplateHTML = `
{{if .IsVideo}}
	<video autoplay muted loop style="width:100%; height:100%; object-fit:contain;" src="{{.MediaSrc}}"></video>
{{else}}
	<img style="width:100%; height:100%; object-fit:contain;" src="{{.MediaSrc}}" alt="media">
{{end}}
`

func main() {
	var err error
	// Load and sort media files for displays A and B.
	filesA, err = loadFiles("./data/a")
	if err != nil {
		log.Fatal("Error loading ./data/a: ", err)
	}
	filesB, err = loadFiles("./data/b")
	if err != nil {
		log.Fatal("Error loading ./data/b: ", err)
	}
	if len(filesA) == 0 || len(filesB) == 0 {
		log.Fatal("No files found in one of the directories.")
	}

	// Route handlers.
	http.HandleFunc("/a", serveA)
	http.HandleFunc("/b", serveB)
	http.HandleFunc("/content/a", contentA)
	http.HandleFunc("/content/b", contentB)
	http.HandleFunc("/advance", advanceHandler)
	http.HandleFunc("/sse", sseHandler)

	// Serve static media files.
	http.Handle("/static/a/", http.StripPrefix("/static/a/", http.FileServer(http.Dir("./data/a"))))
	http.Handle("/static/b/", http.StripPrefix("/static/b/", http.FileServer(http.Dir("./data/b"))))

	port := ":8080"
	log.Println("Server started on port", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// loadFiles returns a sorted list of file names in the specified directory.
func loadFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && isMedia(entry.Name()) {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// serveA renders the full page for Display A.
func serveA(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title      string
		ContentURL string
	}{
		Title:      "Display A",
		ContentURL: "/content/a",
	}
	fullPageTmpl.Execute(w, data)
}

// serveB renders the full page for Display B.
func serveB(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title      string
		ContentURL string
	}{
		Title:      "Display B",
		ContentURL: "/content/b",
	}
	fullPageTmpl.Execute(w, data)
}

// contentA returns the media snippet for Display A.
func contentA(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	idx := currentIndex
	mu.Unlock()
	log.Printf("index: %d/n", idx)
	fileName := filesA[idx%len(filesA)]
	mediaSrc := "/static/a/" + fileName
	data := struct {
		MediaSrc string
		IsVideo  bool
	}{
		MediaSrc: mediaSrc,
		IsVideo:  isVideo(fileName),
	}
	contentTmpl.Execute(w, data)
}

// contentB returns the media snippet for Display B.
func contentB(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	idx := currentIndex
	mu.Unlock()
	fileName := filesB[idx%len(filesB)]
	mediaSrc := "/static/b/" + fileName
	data := struct {
		MediaSrc string
		IsVideo  bool
	}{
		MediaSrc: mediaSrc,
		IsVideo:  isVideo(fileName),
	}
	contentTmpl.Execute(w, data)
}

// isVideo checks if a file extension indicates a video file.
func isVideo(fileName string) bool {
	ext := filepath.Ext(fileName)
	switch ext {
	case ".mp4", ".webm", ".ogg":
		return true
	default:
		return false
	}
}

// advanceHandler increments the global index and broadcasts an SSE event.
func advanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("advanced")
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.Lock()
	currentIndex++
	mu.Unlock()

	// Broadcast the "mediaChanged" event to all connected clients.
	broadcastSSE("mediaChanged")

	w.WriteHeader(http.StatusOK)
}

// sseHandler implements a simple Server-Sent Events endpoint.
func sseHandler(w http.ResponseWriter, r *http.Request) {
	// Set necessary headers for SSE.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Create a channel for this client.
	messageChan := make(chan string, 10)

	// Register the client.
	sseMu.Lock()
	sseClients[messageChan] = true
	sseMu.Unlock()

	// Ensure client removal on disconnect.
	defer func() {
		sseMu.Lock()
		delete(sseClients, messageChan)
		sseMu.Unlock()
	}()

	notify := r.Context().Done()
	for {
		select {
		case msg := <-messageChan:
			fmt.Fprintf(w, "event: %s\n", msg)
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-notify:
			return
		}
	}
}

// broadcastSSE sends the specified message to all connected SSE clients.
func broadcastSSE(message string) {
	sseMu.Lock()
	defer sseMu.Unlock()
	for ch := range sseClients {
		select {
		case ch <- message:
		default:
			// If the client's channel is full, skip sending.
		}
	}
}
func isMedia(fileName string) bool {
	switch filepath.Ext(fileName) {
	case ".jpg", ".png", ".jpeg":
		return true
	case ".mp4", ".webm", ".ogg":
		return true
	default:
		return false
	}
}
