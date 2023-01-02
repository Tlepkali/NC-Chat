package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var joinMessageForUser = []string{
	"Welcome to TCP-Chat!\n",
	"         _nnnn_\n",
	"        dGGGGMMb\n",
	"       @p~qp~~qMb\n",
	"       M|@||@) M|\n",
	"       @,----.JM|\n",
	"      JS^\\__/  qKL\n",
	"     dZP        qKRb\n",
	"    dZP          qKKb\n",
	"   fZP            SMMb\n",
	"   HZM            MMMM\n",
	"   FqM            MMMM\n",
	" __| \".        |\\dS\"qML\n",
	" |    `.       | `' \\Zq\n",
	"_)      \\.___.,|     .'\n",
	"\\____   )MMMMMP|   .'\n",
	"     `-'       `--'\n",
	"[ENTER YOUR NAME]: ",
}

var (
	clients           = make(map[string]net.Conn)
	leaving           = make(chan message)
	messages          = make(chan message)
	historyOfMessages = [][]string{}
	userCounter       int
)

type message struct {
	time     string
	userName string
	text     string
	address  string
}

func main() {
	port := "8989"
	if len(os.Args) != 1 {
		var err error
		port, err = portChecker()
		if err != nil {
			log.Println("[USAGE]: ./TCPChat $port")
		}
	}
	fmt.Println("TCP Server listening on port:", port)
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println("[USAGE]: ./TCPChat $port")
	}
	defer listen.Close()
	var mutex sync.Mutex
	go broadcaster(&mutex)
	for {
		// fmt.Println(len(clients))
		conn, err := listen.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(conn, &mutex)
	}
}

func portChecker() (string, error) {
	if len(os.Args) != 2 {
		err := errors.New("Too many arguments")
		return "", err
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return "", err
	}
	if port < 1024 || port > 65535 {
		err = errors.New("The port numbers in the range 0-1023 are system ports which are reserved for system services.\nThe allowed ports for your server are between 1024-65535, try entering a number in that range.")
		return "", err
	}
	return os.Args[1], nil
}

func addUser(mutex *sync.Mutex) {
	mutex.Lock()
	userCounter++
	mutex.Unlock()
}

func handle(conn net.Conn, mutex *sync.Mutex) {
	mutex.Lock()
	if userCounter >= 10 {
		mutex.Unlock()
		fmt.Fprintln(conn, "Number of user is 10, comeback later :)")
		conn.Close()
		return
	}
	mutex.Unlock()
	go addUser(mutex)
	for _, s := range joinMessageForUser {
		fmt.Fprint(conn, s)
	}
	user := userExist(conn, mutex)
	for _, s := range historyOfMessages {
		for _, historyOfMessagesString := range s {
			fmt.Fprint(conn, historyOfMessagesString)
		}
	}
	if user != "" {
		messages <- newMessage(user, " has joined our chat...", conn)
	}
	input := bufio.NewScanner(conn)
	for input.Scan() {
		messages <- newMessage(user, input.Text(), conn)
	}
	// Delete client from map
	mutex.Lock()
	delete(clients, user)
	userCounter--
	mutex.Unlock()
	if user != "" {
		leaving <- newMessage(user, " has left our chat...", conn)
	}
	conn.Close()
}

func userNameCorrect(name string, conn net.Conn) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9_-]*$", name)
	if !match {
		fmt.Fprint(conn, "User name invalid\n"+"Use only Latin letter with numbers and [-_]\n"+"[ENTER YOUR NAME]: ")
	}
	return match
}

func userExist(conn net.Conn, mutex *sync.Mutex) string {
	userName := ""
	userExist := true
	for userExist {
		scanner := bufio.NewScanner(conn)
		scanner.Scan()
		if userNameCorrect(scanner.Text(), conn) {
			exist := false
			mutex.Lock()
			for k := range clients {
				if k == scanner.Text() {
					exist = true
					fmt.Fprint(conn, "Username already exist\n"+"[ENTER YOUR NAME]: ")
					break
				}
			}
			if !exist {
				clients[scanner.Text()] = conn
				userName = scanner.Text()
				userExist = false
			}
			mutex.Unlock()
		}
	}
	return userName
}

func newMessage(user string, msg string, conn net.Conn) message {
	addr := conn.RemoteAddr().String()
	msgTime := "[" + time.Now().Format("01-02-2006 15:04:05") + "]"
	if msg != " has joined our chat..." && msg != " has left our chat..." && msg != "" {
		temp := []string{msgTime, "[" + user + "]", msg, "\n"}
		historyOfMessages = append(historyOfMessages, temp)
	}
	return message{
		time:     msgTime,
		userName: user,
		text:     msg,
		address:  addr,
	}
}

func checkValidMassage(text string) bool {
	if text == "" {
		return false
	}
	for _, char := range text {
		if char < 32 || char > 126 {
			return false
		}
	}
	return true
}

func broadcaster(mutex *sync.Mutex) {
	for {
		select {
		case msg := <-messages:
			mutex.Lock()
			for k, conn := range clients {
				var validMassage bool
				validMassage = checkValidMassage(msg.text)
				if msg.address == conn.RemoteAddr().String() {
					if msg.text == " has joined our chat..." {
						time.Sleep(time.Millisecond * 300)
					}
					fmt.Fprint(conn, msg.time+"["+msg.userName+"]"+":")
					continue
				}
				if msg.text == " has joined our chat..." {
					fmt.Fprintln(conn, "\n"+msg.userName+msg.text)
					fmt.Fprint(conn, msg.time+"["+k+"]"+":")
					continue
				}
				if validMassage {
					fmt.Fprintln(conn, "\n"+msg.time+"["+msg.userName+"]"+":"+msg.text)
					fmt.Fprint(conn, msg.time+"["+k+"]"+":")
				}
			}
			mutex.Unlock()
		case msg := <-leaving:
			mutex.Lock()
			for k, conn := range clients {
				fmt.Fprintln(conn, "\n"+msg.userName+msg.text)
				fmt.Fprint(conn, msg.time+"["+k+"]"+":")
			}
			mutex.Unlock()
		}
	}
}
