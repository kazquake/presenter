package main

import (
	"embed"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	connections []*websocket.Conn
	connMutex   sync.Mutex
	station     string
	rsLinkEmbed string
)

func main() {
	udpPort := getEnv("UDP_PORT", "5005")
	udpIP := getEnv("UDP_IP", "0.0.0.0")
	webPort := getEnv("WEB_PORT", "8080")
	station = getEnv("STATION", "R1B7B")
	rsLinkEmbed = getEnv("RS_LINK_EMBED", "https://dataview.raspberryshake.org/#/embed/AM/R1B7B/00/EHZ")

	http.Handle("/static/", http.StripPrefix("", http.FileServer(http.FS(staticFiles))))

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", htmlHandler("static/index.html"))
	http.HandleFunc("/alert", htmlHandler("static/alert.html"))
	http.HandleFunc("/more", htmlHandler("static/more.html"))

	go udpListener(udpIP, udpPort)

	log.Println("CONTACTS: Rishat Sultanov www.rishat.space")
	log.Println("Listening on port", webPort)

	log.Fatal(http.ListenAndServe(":"+webPort, nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	connMutex.Lock()
	connections = append(connections, conn)
	connMutex.Unlock()
}

func udpListener(ip, port string) {
	addr := net.UDPAddr{
		Port: atoi(port),
		IP:   net.ParseIP(ip),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		data := string(buffer[:n])
		connMutex.Lock()
		for i := 0; i < len(connections); i++ {
			err := connections[i].WriteMessage(websocket.TextMessage, []byte(data))
			if err != nil {
				log.Println("Write error:", err)
				connections[i].Close()
				connections = append(connections[:i], connections[i+1:]...)
				i-- // Adjust index after removing an element
			}
		}
		connMutex.Unlock()
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func atoi(str string) int {
	value, err := strconv.Atoi(str)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

func replaceStation(html, station string) string {
	return strings.ReplaceAll(html, "{{STATION}}", station)
}

func replaceRSLinkEmbed(html, rsLink string) string {
	return strings.ReplaceAll(html, "{{RS_LINK_EMBED}}", rsLink)
}

func htmlHandler(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFiles.ReadFile(filePath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		html := string(data)
		html = replaceStation(html, station)
		html = replaceRSLinkEmbed(html, rsLinkEmbed)
		w.Write([]byte(html))
	}
}
