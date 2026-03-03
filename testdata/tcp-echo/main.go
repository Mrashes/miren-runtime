package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	// TCP echo server on port 7000
	go func() {
		ln, err := net.Listen("tcp", ":7000")
		if err != nil {
			log.Fatal("tcp listen:", err)
		}
		log.Println("TCP echo server listening on :7000")
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println("tcp accept:", err)
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	// HTTP health endpoint on port 3000
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	log.Printf("HTTP health server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
