package main

import "net"
import "fmt"
import "bufio"
import "time"
import "strings"
import "container/list"
import "regexp"
import "strconv"
import "encoding/binary"
import "os"
//import "io"

type IRC struct {
    Server string
    Channel string
    Nick string
    conn net.Conn
    ch chan IRCMsg
    quit chan bool // channel quits every running goroutine
}

type IRCMsgType int

const (
    SUBSCRIBE IRCMsgType = iota
    SUBSCRIBEMSG
    SENDCMD
    MSGRECV)

type IRCMsg struct {
    msgType IRCMsgType
    msgContent string
    backchannel chan IRCMsg
}

func (i *IRC) PingListener(channel chan IRCMsg) {
    for {
        select {
        case ping := <-channel:
            i.SendCommand("PONG " + strings.Split(ping.msgContent, " ")[1])
        case <-i.quit:
            return
        }
    }
}
func (i *IRC) AddSubscription(filter string, bch chan IRCMsg, once bool) {
    i.ch<-IRCMsg{SUBSCRIBE, filter, bch}
}
func (i *IRC) AddSubscriptionForMessage(filter string, bch chan IRCMsg, once bool) {
    i.ch<-IRCMsg{SUBSCRIBEMSG, filter, bch}
}
func (i *IRC) SendCommand(command string) {
    fmt.Print("send cmd " + command)
    i.ch<-IRCMsg{SENDCMD, command, nil}
}
func (i *IRC) Connect() bool {
    // setup quit channel
    i.quit = make(chan bool)
    // setup channel for comm handler
    i.ch = make(chan IRCMsg, 1000)

    // setup pingponger
    ppChan := make(chan IRCMsg, 10)
    go i.PingListener(ppChan)
    i.AddSubscription("PING", ppChan, false)


    // conenct to server
    conn, _ := net.Dial("tcp", i.Server)
    i.conn = conn
    fmt.Fprintf(conn, "NICK %s\r\n", i.Nick)
    fmt.Fprintf(conn, "USER %s 8 * : %s\r\n", i.Nick)
    
    // callback channel for registration complete
    regChan := make(chan IRCMsg)
    i.AddSubscription("MODE", regChan, false)

    go i.ConnHandler(i.ch)

    select {
        case <-regChan:
        case <-time.After(4*time.Second):
            fmt.Print("Timeout")
            close(i.quit)
            return false
    }
    
    return i.JoinChannel(i.Channel)
}
func (i *IRC) JoinChannel(channel string) bool {
    joinedAwait := make(chan IRCMsg)
    i.AddSubscription("JOIN", joinedAwait, false)
    i.SendCommand("JOIN " + channel)
    select {
        case <-joinedAwait:
        case <-time.After(4*time.Second):
            i.Quit()
            return false
    }
    fmt.Println("joined!!")
    return true



}
func (i *IRC) ConnHandler(ch chan IRCMsg) {
    privmsgRegexp := regexp.MustCompile(`PRIVMSG ([^ ]+) :(.*)`)
    noprefixRegexp := regexp.MustCompile(`^:[^ ]+ (.*)`)
    subscriptions := list.New()
    readCh := make(chan string, 10)
    reader := bufio.NewReader(i.conn)
    go func(ch chan string) {
        for {
            msg, _ := reader.ReadString('\n')
            ch <- msg
        }
    }(readCh)

    for {
        select {
        case <-i.quit:
            fmt.Println("quit")
            return
        case cmd := <-ch:
            // handle incoming comm requests
            switch cmd.msgType {
            case SUBSCRIBEMSG:
                subscriptions.PushBack(cmd)
            case SUBSCRIBE:
                subscriptions.PushBack(cmd)
            case SENDCMD:
                fmt.Fprintf(i.conn, cmd.msgContent + "\r\n")
            }

        case msg := <-readCh:
            // remove prefix
            //ffmt.Print(msg)

            msg = noprefixRegexp.ReplaceAllString(msg, "${1}")

            if strings.Contains(msg, "PRIVMSG someRandom") {
                fmt.Print(msg)
                fmt.Println(subscriptions)
            }

            // handle incoming message from irc server
            for el := subscriptions.Front(); el != nil; el = el.Next() {
                elVal := el.Value.(IRCMsg)
                switch elVal.msgType {

                    case SUBSCRIBEMSG:
                        // if recipient matches filter string, forward message

                        if (privmsgRegexp.MatchString(msg) &&
                    strings.Contains(privmsgRegexp.FindStringSubmatch(msg)[1], elVal.msgContent)) {
                            fmt.Println("rule struck: %s, msg: %s", elVal.msgContent, msg)
                            elVal.backchannel <- IRCMsg{MSGRECV, privmsgRegexp.FindStringSubmatch(msg)[2], nil}
                        }

                    case SUBSCRIBE:
                        if (strings.HasPrefix(msg, elVal.msgContent)) {
                            fmt.Println("rule struck: %s, msg: %s", elVal.msgContent, msg)
                            elVal.backchannel <- IRCMsg{MSGRECV, msg, nil}
                        }
                }
            }

        }
    }
}
func (i *IRC) Quit() {
    fmt.Print("Quit")
    close(i.quit)
}
func (i *IRC) RequestPackage(bot string, packageId string) (bool){
    var feedback IRCMsg
    recvFeedback := false
    awaitFeedback := make(chan IRCMsg)
    i.AddSubscriptionForMessage(i.Nick, awaitFeedback, true)
    F:
    for a:=0; a<3; a++ {
        // await a random private message to myself - todo: must be by the bot

        i.SendCommand(fmt.Sprintf("PRIVMSG %s :xdcc send %s", bot, packageId))

        select {
            case feedback = <-awaitFeedback:
                recvFeedback = true
                break F
            case <-time.After(15 * time.Second):
        }
    }
    if !recvFeedback {
        i.Quit()
        return false
    }
    fmt.Print("got feedback")
    fmt.Print(feedback.msgContent)
    r := regexp.MustCompile("DCC SEND (.+)")
    msg := r.ReplaceAllString(feedback.msgContent, "${1}")
    s := strings.Split(msg, " ")
    filename, ip, port, size := s[0], s[1], s[2], s[3]
    fmt.Printf("File: %s, ip: %s, port: %s, size: %s", filename, ip, port, size)
    a, _ := strconv.Atoi(ip)
    fmt.Println(a)
    ipstr := fmt.Sprintf("%d.%d.%d.%d", byte(a>>24), byte(a>>16), byte(a>>8), byte(a))

    conn, err := net.Dial("tcp", ipstr + ":" + port)
    fmt.Println("connected")
    if err != nil {
        fmt.Println(err)
        // handle error
    }
    recvBuf := make([]byte, 1024)
    f, err := os.Create("/tmp/file")
    defer f.Close()
    G:
        for {
        n, err2 := conn.Read(recvBuf[:]) // recv data
    if err2 != nil {
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            fmt.Println("read timeout:", netErr, n)
            break G
            // time out
        } else {
            fmt.Println("read error:", err2, n)
            break G
            // some error else, do something else, for example create new conn
        }
    }
    f.Write(recvBuf[:n])
    f.Sync()
    //io.CopyN(f, recvBuf, uint64(n))
    bs := make([]byte, 4)
    binary.BigEndian.PutUint32(bs, uint32(n))
    conn.Write(bs)
    fmt.Print(n)
}
f.Close()

    return true
}

func main() {
    i := IRC{Server: "irc.abjects.net:6667", Channel: "#moviegods", Nick: "asdfasafaaffafafafd"}
    i.Connect()
    i.RequestPackage("[MG]-Request|Bot|Bud", "#29")
/*
    i := IRC{Server: "irc.criten.net:6667", Channel: "#ELITEWAREZ", Nick: "asdfasafaaffafafafd"}
    i.Connect()
    i.RequestPackage("[EWG]-[TBABROAD-5", "#67")*/
    for {
        time.Sleep(10000)
    }
}
/*
  // connect to this socket
  conn, _ := net.Dial("tcp", "irc.criten.net:6667")
    // read in input from stdin
    fmt.Print("Text to send: ")
    fmt.Fprintf(conn, "NICK asdaskdljwelknj\r\n")

    fmt.Fprintf(conn, "USER asdasdasd 8 * : asdads\r\n")



    reader := bufio.NewReader(conn)
    state := 0
    for {

	pongRecv := recMsg(conn, reader)

	if (state == 0 && pongRecv) {
		fmt.Print("pong received")
		state++
	time.Sleep(2*time.Second)	
	}
	if (state == 1) {
		fmt.Fprintf(conn, "JOIIN #0DAY-MP3S\r\n")
		state++
	time.Sleep(5*time.Second)	

	}
	if (state == 2) {
		fmt.Fprintf(conn, "PRIVMSG Oday-YEA2 :xdcc send #70\r\n")
		state++
	time.Sleep(2*time.Second)	
	}
    }


}

func recMsg(conn net.Conn, reader *bufio.Reader) bool {
	message,_ := reader.ReadString('\n')
	splitted := strings.Split(message, " ")
	if (splitted[0] == "PING") {
            fmt.Fprintf(conn, "PONG %s", splitted[1])
	    return true
	} else {
		fmt.Print(message + "\n")
	}
	return false
}


*/
