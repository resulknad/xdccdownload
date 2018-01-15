package main

import "net/url"
import "path"
import "net"
import "fmt"
import "time"

//import "strings"
import "regexp"
import "strconv"
import "encoding/binary"
import "os"

type XDCC struct {
	IRCConn *IRC
	Bot     string
	Channel string
	Package string
}

func (i *XDCC) awaitFeedbackAfterRequest(ch chan PrivMsg) (string, bool) {
	select {
	case feedback := <-ch:
		return feedback.Content, true
	case <-time.After(15 * time.Second):
		return "", false
	}
}

func (i *XDCC) Download() bool {
	i.IRCConn.JoinChannel(i.Channel)
	awaitFeedback := make(chan PrivMsg)
	i.IRCConn.SubscriptionCh <- PrivMsgSubscription{To: i.IRCConn.Nick, Backchannel: awaitFeedback, Once: true}

	var feedback string
	recv := false

	feedback, recv = i.awaitFeedbackAfterRequest(awaitFeedback)
	for a := 0; a < 3 && recv == false; a++ {
        fmt.Println("trying...")
		i.IRCConn.CommandCh <- fmt.Sprintf("PRIVMSG %s :xdcc send %s", i.Bot, i.Package)
		feedback, recv = i.awaitFeedbackAfterRequest(awaitFeedback)
	}

	r := regexp.MustCompile(`DCC SEND ((?:"[^"]+")|(?:[^ ]+)) ([0-9]*) ([0-9]*) ([0-9]*)`)

	if (recv == false) || (!r.MatchString(feedback)) {
		fmt.Println("no feedback from bot. or not dcc send")
		i.IRCConn.Quit()
		return false
	}

	fmt.Print("got feedback")
	fmt.Print(feedback)

	s := r.FindStringSubmatch(feedback)
	filename, ip, port, size := s[1], s[2], s[3], s[4]
	fmt.Printf("File: %s, ip: %s, port: %s, size: %s", filename, ip, port, size)
	a, _ := strconv.Atoi(ip)
	fmt.Println(a)
	ipstr := fmt.Sprintf("%d.%d.%d.%d", byte(a>>24), byte(a>>16), byte(a>>8), byte(a))

	sizeI, _ := strconv.Atoi(size)
	conn, err := net.Dial("tcp", ipstr+":"+port)
	fmt.Println("connected")
	if err != nil {
		fmt.Println(err)
		return false
	}
	var recvBytes int
    var recvBytesSinceLastAck int
	recvBytes = 0
    recvBytesSinceLastAck = 0
	recvBuf := make([]byte, 4096)
	f, err := os.Create(path.Join("/home/dan/Downloads/", url.PathEscape(filename)))
	defer f.Close()
G:
	for {
		n, err2 := conn.Read(recvBuf[:]) // recv data
		if err2 != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("read timeout:", netErr, n)
			} else {
				fmt.Println("read error:", err2, n)
			}
            if recvBytes != sizeI && recvBytes <= (1<<32 - 1) {
                bs := make([]byte, 4)
                binary.BigEndian.PutUint32(bs, uint32(recvBytesSinceLastAck))
                conn.Write(bs)
            } else {
                break G
            }
		}
		recvBytes = recvBytes + n
        recvBytesSinceLastAck += n
		f.Write(recvBuf[:n])

		//io.CopyN(f, recvBuf, uint64(n))

		fmt.Println(recvBytes, " / ", size)
		if recvBytes == sizeI {
			fmt.Println("Received file.")
			break G
		}
	}
    f.Sync()
	f.Close()

	return  recvBytes == sizeI
}

