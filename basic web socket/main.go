package main

import "github.com/gorilla/websocket"
import "fmt"
import "net/http"

type Person struct {
	Name   string `json:"username"`
	Alamat string `json:"alamat"`
}

var upgrader = websocket.Upgrader{}

func halaman_index(res http.ResponseWriter, req *http.Request) {

	http.ServeFile(res, req, "index.html")

}

func write_message(res http.ResponseWriter, req *http.Request) {
	var conn, _ = upgrader.Upgrade(res, req, nil)
	go func(conn *websocket.Conn) {
		for {
			mytype, msg, _ := conn.ReadMessage()

			conn.WriteMessage(mytype, msg)
		}

	}(conn)
}

func read_message(res http.ResponseWriter, req *http.Request) {
	var conn, _ = upgrader.Upgrade(res, req, nil)
	go func(conn *websocket.Conn) {
		for {
			_, msg, _ := conn.ReadMessage()

			fmt.Println(string(msg))
		}

	}(conn)
}

func send_message(res http.ResponseWriter, req *http.Request) {
	var conn, _ = upgrader.Upgrade(res, req, nil)
	go func(conn *websocket.Conn) {
		for {
			_, msg, _ := conn.ReadMessage()
			fmt.Println(string(msg))

			conn.WriteJSON(Person{
				Name:   "Reno Syahputra",
				Alamat: "Jln Janti",
			})
		}

	}(conn)
}

func main() {
	http.HandleFunc("/", halaman_index)
	http.HandleFunc("/v1/ws", write_message)
	http.HandleFunc("/v2/ws", read_message)
	http.HandleFunc("/v3/ws", send_message)
	fmt.Println("running now...")
	http.ListenAndServe(":8080", nil)
}
