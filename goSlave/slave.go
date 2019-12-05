package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var origChunks string
var repChunks string

func heartbeat(c net.Conn) {

	for {

		time.Sleep(10000)
		origChunks, repChunks = initChunks("/home/arbaz/password/original", "/home/arbaz/password/replica")
		c.Write([]byte("Alive" + ":" + origChunks + "-" + repChunks + "\n"))
		// c.Write([]byte("-Alive"))
		time.Sleep(10000)
	}
}
func serachFile(password string, fileName string) (int, string) {
	file, err := os.Open("/home/arbaz/password/original/" + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// test := strings.TrimSpace(password)
	// log.Printf("Search: %v", []byte(test))
	// log.Printf("SearchO: %v", []byte("030887"))

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		if scanner.Text() == password {
			log.Println("FOUND.")
			return 1, fileName
		}
	}
	return 0, fileName
}
func search(conn net.Conn, password string, origChunks string, repChunks string) {

	log.Println("Starting Search")
	files := strings.Split(origChunks, ",")
	flag := 0
	go func() {
		message := make([]byte, 1092)
		n, _ := conn.Read(message)
		log.Printf("Value recv:  %v", message[:n])
		if string(message[:n]) == "stop" {
			flag = 1
		}
	}()
	for i := 0; i < len(files); i++ {
		if flag == 0 {
			log.Printf("Searching Chunk: %v", files[i])
			found, chunk := serachFile(password, files[i])
			if found == 1 {
				log.Println("Send to Server")
				// Server Sending Code
				conn.Write([]byte("found:" + chunk))
				return
			}

		} else {
			log.Printf("Stopping. Already Found by my comrade.")
			return
		}
	}
	conn.Write([]byte("not found:"))
}
func requestHandle(c net.Conn) {
	buf := make([]byte, 4096)
	n, _ := c.Read(buf)
	log.Printf("Password: %s", string(buf))
	passwordToSearch := string(buf[:n])
	search(c, passwordToSearch, origChunks, repChunks)
}

func getCommand(serverPort string) {

	log.Printf("Listening for Server Request on %s", serverPort)
	ln, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("Server Connected.")
		go requestHandle(conn)
	}

}

func initConnection(ip string, serverPort string, listenPort string) {

	conn, err := net.Dial("tcp", ip+":"+serverPort)
	if err != nil {
		log.Fatal(err)
	}
	//defer conn.Close()

	origChunks, repChunks = initChunks("/home/arbaz/password/original", "/home/arbaz/password/replica")

	log.Println(origChunks)
	log.Println(repChunks)
	// conn.Write([]byte("1,2,3,4,5,6,10" + ":" + listenPort))
	conn.Write([]byte(listenPort + ":" + origChunks + "-" + repChunks))
	// conn.Write([]byte(orig))
	// conn.Write([]byte(replica))
	go heartbeat(conn)

}
func initChunks(origPath string, repPath string) (string, string) {

	var original string
	var replicas string

	/*Original Chunks*/
	files, err := ioutil.ReadDir(origPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		// original = append(original, file.Name())
		original = original + file.Name() + ","
		// fmt.Println(file.Name())
	}

	/*Replicated Chunks*/
	files, err = ioutil.ReadDir(repPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		// replicas = append(replicas, file.Name())
		replicas = replicas + file.Name() + ","
		// fmt.Println(file.Name())
	}
	return original, replicas
}

func main() {

	args := os.Args[1:]
	ip := args[0]
	serverPort := args[1]
	listenPort := args[2]
	blockChan := make(chan int)

	initConnection(ip, serverPort, listenPort)
	go getCommand(listenPort)

	log.Printf(string(<-blockChan))

}
