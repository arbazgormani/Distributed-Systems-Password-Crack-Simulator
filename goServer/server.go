package main

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

//Client Structure
type Client struct {
	connection *http.Request
	password   string
}

//Slave Structure
type Slave struct {
	connection     net.Conn
	port           string
	origChunks     string
	repChunks      string
	chunksToSearch string
}

//Page represents a Page
type Page struct {
	Title string
	Text  string
	MSG   string
	MSGN  string
}

var clientChannel = make(chan Client)
var numSlaves = 0
var passFound = 0

// For Web Client
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	editTemplate, _ := template.ParseFiles("index.html")
	loadedPage := &Page{Title: "Password Crack"}
	if numSlaves == 0 {
		loadedPage = &Page{Title: "Password Crack", Text: "Slaves are not Present"}
	}
	if passFound == 1 {
		loadedPage = &Page{Title: "Password Crack", MSG: "Yes! Found the Password"}
		passFound = 0
	}
	editTemplate.Execute(w, &loadedPage)
}
func searchHandler(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	log.Printf(password)
	newClient := Client{connection: r, password: string(password)}
	clientChannel <- newClient

	for {
		// log.Printf(string(passFound))
		if passFound == 1 {
			http.Redirect(w, r, "/", http.StatusFound)
			break
		}
	}

}

// Initial Client Handler
// func clientHandle(c net.Conn, addchan chan Client) {

// 	c.Write([]byte("Enter Password to Search: "))
// 	clientReader := bufio.NewReader(c)
// 	password, _, _ := clientReader.ReadLine()
// 	newClient := Client{connection: c, password: string(password)}
// 	addchan <- newClient
// }

//Initial Slave Handler
func slaveHandle(c net.Conn, addchan chan Slave, discon chan Slave) {

	log.Printf("Slave Connected :)")
	port := ""
	buf := make([]byte, 4096)
	// n, _ := c.Read(buf)
	// dummy := string(buf)
	n, _ := c.Read(buf)
	info := string(buf)
	idx := strings.Index(info, ":")
	port = info[0:idx]
	idx2 := strings.Index(info, "-")
	// Subtract 1 to remove comma
	origChunks := info[idx+1 : idx2-1]
	repChunks := info[idx2+1 : n-1]
	// idx := strings.Index(dummy, ":")
	// chunk := dummy[:idx]

	// idx2 := strings.Index(dummy, "-")
	// if idx2 != -1 {
	// 	port = dummy[idx:idx2]
	// } else {
	// 	port = dummy[idx : idx+n-len(chunk)]
	// }
	numSlaves++

	newSlave := Slave{connection: c, origChunks: origChunks, repChunks: repChunks, port: port}
	addchan <- newSlave

	heartBuff := make([]byte, 4096)
	for {
		n, err := c.Read(heartBuff)
		if err != nil || n == 0 {
			c.Close()
			break
		} else {
			info = string(heartBuff)
			n = strings.Index(info, "\n")
			newInfo := info[0:n]
			idx = strings.Index(newInfo, ":")
			idx2 = strings.Index(newInfo, "-")
			if idx != -1 && idx2 != -1 {
				// Subtract 1 to remove comma
				origChunks = newInfo[idx+1 : idx2-1]
				repChunks = newInfo[idx2+1 : n-1]
				newSlave.origChunks = origChunks
				newSlave.repChunks = repChunks
				addchan <- newSlave
				heartBuff = make([]byte, 4096)

			}

		}
		// log.Printf("Status: %d", n)
		// if string(heartBuff) == string(buf) {
		// 	log.Printf("Same Chunks.")
		// } else {
		// 	log.Printf("Chunks Changed.")
		// }
		// heartBuff = heartBuff[:0]
	}

	defer func() {
		log.Printf("Connection from %v closed.\n", c.RemoteAddr())
		discon <- newSlave
		numSlaves--
	}()
}

func sendToSlave(ip string, port string, password string, newslaves chan Slave, foundChan chan string, notfound chan string) {

	log.Printf("Con: %v", ip+":"+port)
	// log.Println
	// log.Printf("port: %v", []byte(":4500"))
	// log.Printf("hc: %v", []byte(port))

	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		log.Fatal(err)
	}
	buff := make([]byte, 1092)

	dummySlave := Slave{connection: conn, origChunks: "Nill", repChunks: "Nill", port: port}
	newslaves <- dummySlave

	conn.Write([]byte(password))
	n, _ := conn.Read(buff)
	message := string(buff[:n])
	idx := strings.Index(message, ":")
	if idx != -1 {
		msg := message[:idx]
		if msg == "found" {
			foundChan <- message
		} else {
			notfound <- message
		}
	}

}

//Schedule search
func scheduleSearch(slaveChan chan Slave, clientChannel chan Client, rmchan chan Slave) {
	slaves := make(map[net.Conn]Slave)
	newslaves := make(map[net.Conn]Slave)
	clients := make(map[*http.Request]Client)

	foundChan := make(chan string)
	notfound := make(chan string)
	newslaveChan := make(chan Slave)

	// chunks := "0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19"

	// slaves := make([]Slave, 0, 20)
	// clients := make([]Client, 0, 20)

	for {
		select {
		case newSlave := <-slaveChan:
			// log.Printf("Slave Connected. Original Chunks: %s", newSlave.origChunks)
			// log.Printf("Slave Connected. Replicated Chunks: %s", newSlave.repChunks)
			// log.Printf("port: %s", newSlave.port)
			// log.Printf("ip: %s", newSlave.connection.RemoteAddr().String())

			// log.Printf("IP: %s", newSlave.connection.RemoteAddr().String())
			// slaves = append(slaves, newSlave)
			slaves[newSlave.connection] = newSlave

		case newClient := <-clientChannel:
			log.Printf("Client Connected. Password to search: %s", newClient.password)
			// clients = append(clients, newClient)
			clients[newClient.connection] = newClient
			password := newClient.password

			for k, v := range slaves {
				// passFound = 0
				log.Println(k)
				ip := v.connection.RemoteAddr().String()
				ip = ip[:strings.Index(ip, ":")]
				log.Println(ip)
				log.Println(v.port)
				go sendToSlave(ip, v.port, password, newslaveChan, foundChan, notfound)
				log.Printf("IP: %s", v.connection.RemoteAddr().String())
			}

		case dummy := <-newslaveChan:
			newslaves[dummy.connection] = dummy

		case rclient := <-rmchan:
			log.Printf("Slave disconnects: %s\n", rclient.connection.RemoteAddr().String())
			delete(slaves, rclient.connection)
		case <-foundChan:
			/// Key is Connection identifier
			log.Printf("Telling slaves Password is found......")
			log.Printf("Length: %v", len(newslaves))
			for key := range newslaves {
				key.Write([]byte("stop"))
				// log.Printf(message)
				passFound = 1
			}
			newslaves = make(map[net.Conn]Slave)

		}

		// log.Printf("len: %d", len(slaves))
		// log.Println("Slaves: ", slaves)
	}

}

//Starts listening for Client
// func startClient(clientPort string, clientChannel chan Client) {

// 	ln, err := net.Listen("tcp", ":"+clientPort)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	for {
// 		conn, err := ln.Accept()
// 		if err != nil {
// 			log.Println(err)
// 			continue
// 		}
// 		go clientHandle(conn, clientChannel)
// 	}
// }

//Starts listening for Slaves
func startSlave(slavePort string, slaveChannel chan Slave, discon chan Slave) {
	ln, err := net.Listen("tcp", ":"+slavePort)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		// log.Printf("Slave Connected.")
		go slaveHandle(conn, slaveChannel, discon)
	}
}

func main() {

	args := os.Args[1:]
	clientPort := ""
	slavePort := ""

	/*Communication Channels*/
	// clientChannel := make(chan Client)
	slaveChannel := make(chan Slave)
	disconChannel := make(chan Slave)
	// commChannel := make(chan string)

	blockChannel := make(chan int)

	// slave_comm := make(chan string)

	if len(args) == 1 {
		clientPort = os.Args[1]
		slavePort = "9999"
	} else if len(args) == 2 {
		clientPort = os.Args[1]
		slavePort = os.Args[2]
	} else {
		clientPort = "8888"
		slavePort = "9999"

	}

	fmt.Println("Client Port: ", clientPort)
	fmt.Println("Slave Port: ", slavePort)

	go scheduleSearch(slaveChannel, clientChannel, disconChannel)
	//go startClient(clientPort, clientChannel)
	go startSlave(slavePort, slaveChannel, disconChannel)

	// Web Part
	go http.HandleFunc("/", defaultHandler)
	go http.HandleFunc("/search/", searchHandler)
	go http.ListenAndServe(":"+clientPort, nil)

	log.Printf(string(<-blockChannel))
}
