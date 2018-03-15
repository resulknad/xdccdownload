package main

import "net"
import "fmt"
import "bufio"
import "time"
import "strings"
import "container/list"
import "regexp"
import "github.com/elgs/gostrgen"
import "sync"
import "log"
//import "os"

//import "io"

type IRC struct {
    sync.Mutex
    Server string
    Channel string
    Nick string
    conn net.Conn
    SubscriptionCh chan Subscription
    CommandCh chan string
    channels []string

    quit chan bool // channel quits every running goroutine
}

type IRCMsgType int

const (
    SUBSCRIBE IRCMsgType = iota
    SUBSCRIBEMSG
    SENDCMD
    MSGRECV)

type Subscription interface {
    Evaluate(string, *IRC) bool
    GetOnce() bool
}

type CodeSubscription struct {
    Subscription
    Once bool
    Backchannel chan string
    Code string
}

type PrivMsgSubscription struct {
    Subscription
    Once bool
    Backchannel chan PrivMsg
    From string
    To string
    Message string
}
type PrivMsg struct {
    Content string
    From string
    To string
    Server string
    Channel string
}

func (f PrivMsgSubscription) Evaluate(msg string, i *IRC) bool {
    receiverRegexp := regexp.MustCompile(`:([^!]*)!`)
    privmsgRegexp := regexp.MustCompile(`PRIVMSG ([^ ]+) :(.*)`)
    if (privmsgRegexp.MatchString(msg) &&
        strings.Contains(privmsgRegexp.FindStringSubmatch(msg)[1], f.To)) {
            //fmt.Println("rule struck: %s, msg: %s, channel:%s", f.To, msg, i.Channel)
            f.Backchannel <- PrivMsg{Content: privmsgRegexp.FindStringSubmatch(msg)[2], From: receiverRegexp.FindStringSubmatch(msg)[1], Server: i.Server, Channel: i.Channel, To: privmsgRegexp.FindStringSubmatch(msg)[1]}
        return true
    }
    return false
}

func (f PrivMsgSubscription) GetOnce() bool {
    return f.Once;
}

type GeneralSubscription struct {
    Subscription
    Once bool
    Backchannel chan string
    Filter string
}

func removePrefix(msg string) string {
    noprefixRegexp := regexp.MustCompile(`^:[^ ]+ (.*)`)
    return noprefixRegexp.ReplaceAllString(msg, "${1}")
}

func removeWholePrefix(msg string) string {
	noprefixRegexp := regexp.MustCompile(`^:[^ ]+[^:]+:`)
    return noprefixRegexp.ReplaceAllString(msg, "${1}")
}

func (f GeneralSubscription) Evaluate(msg string, i *IRC) bool {
	msg = removePrefix(msg)
    if (strings.HasPrefix(msg, f.Filter)) {
        //fmt.Println("rule struck: %s, msg: %s", f.Filter, msg)
        f.Backchannel <- msg
        return true
    }
    return false
}

func (f CodeSubscription) GetOnce() bool {
    return f.Once;
}

func (f CodeSubscription) Evaluate(msg string, i *IRC) bool {
	codeRegexp := regexp.MustCompile(`^:[^ ]+ ([0-9]+)`)


    if codeRegexp.MatchString(msg) &&
        codeRegexp.FindStringSubmatch(msg)[1] == f.Code {
        //fmt.Println("rule struck: %s, msg: %s", f.Filter, msg)
	        f.Backchannel <- removeWholePrefix(msg)
        return true
    }
    return false
}

func (f GeneralSubscription) GetOnce() bool {
    return f.Once;
}

func (i *IRC) PingListener(channel chan string){
    for {
        select {
        case ping := <-channel:
            i.CommandCh <- ("PONG " + strings.Split(ping, " ")[1])
        case <-i.quit:
            return
        }
    }
}

func (i *IRC) Connect() bool {
    if i.Nick == "" {
        str, _ := gostrgen.RandGen(15, gostrgen.Lower | gostrgen.Upper, "", "")
        i.Nick = str
        log.Print("Generated Nickname, ", i.Nick)
    }
    // setup quit channel
    i.quit = make(chan bool)
    // setup channel for comm handler
    i.SubscriptionCh = make(chan Subscription, 1000)
    i.CommandCh = make(chan string, 1000)

    // setup pingponger
    ppChan := make(chan string, 10)
    go i.PingListener(ppChan)
    i.SubscriptionCh<-GeneralSubscription{Once: false, Backchannel: ppChan, Filter: "PING"}

    // conenct to server
    conn, err := net.Dial("tcp", i.Server)
    if err != nil {
        log.Print(err)
        return false
    }
    i.conn = conn
    fmt.Fprintf(conn, "NICK %s\r\n", i.Nick)
    fmt.Fprintf(conn, "USER %s 8 * : %s\r\n", i.Nick, i.Nick)
    
    // callback channel for registration complete
    regChan := make(chan string)
    i.SubscriptionCh<-GeneralSubscription{Once: true, Backchannel: regChan, Filter: "MODE"}

    go i.ConnHandler()

    select {
        case <-regChan:
        case <-time.After(10*time.Second):
            fmt.Print("Timeout")
            close(i.quit)
            return false
    }
    
    return true
}
func (i *IRC) CheckChannel(channel string) bool {
	log.Print("check channel")
    i.Lock()
    defer i.Unlock()
	c:=channel
		// issue names command, which returns users in channel
		// check if we are in this channel
		namesCh := make(chan string, 10)
		i.SubscriptionCh<-CodeSubscription{Once: true, Backchannel: namesCh, Code: "353"}

		i.CommandCh<-("NAMES "+c+"\r\n")
		select {
		case list := <-namesCh:
			if !strings.Contains(list, i.Nick) {
				log.Print("not good")
				return false
			}
		case <-time.After(2*time.Second):
			log.Print("not good")
			return false
		}

	log.Print("good")
	return true
}
func (i *IRC) JoinChannel(channel string) bool {
    i.Lock()
    defer i.Unlock()
    for _,c := range(i.channels) {
        if channel == c {
            return true
        }
    }

    joinedAwait := make(chan string)
    i.SubscriptionCh<-GeneralSubscription{Filter: "JOIN", Backchannel:joinedAwait, Once:true}
    i.CommandCh<-"JOIN " + channel + "\r\n"

    select {
        case <-joinedAwait:
        case <-time.After(10*time.Second):
            log.Print("Join timeout")
            i.Quit()
            return false
    }
    i.Channel = channel
    log.Print("joined!!")
    i.channels = append(i.channels, channel)
    return true
}

func (i *IRC) ConnHandler() {
    subscriptions := list.New()
    readCh := make(chan string, 10)
    reader := bufio.NewReader(i.conn)
    go func(ch chan string) {
        for {

            msg, _ := reader.ReadString('\n')
			if len(msg) > 0 {
			//log.Print(msg)
		}
      //      f, err := os.OpenFile("/tmp/" + i.Nick + ".log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
      //      if err!=nil {
      //          panic(err)
      //      }
      //      f.WriteString(msg + "\n")
      //      f.Close()
            ch <- msg
        }
    }(readCh)

    for {
        select {
        case <-i.quit:
            log.Print("Quitting")
            i.conn.Close()
            return
        case cmd := <-i.CommandCh:
            fmt.Fprintf(i.conn, cmd + "\r\n")
            log.Print(cmd + " sent")
        case filter :=<-i.SubscriptionCh:
            subscriptions.PushBack(filter)
        case msg := <-readCh:
            var next *list.Element
            // handle incoming message from irc server
            for el := subscriptions.Front(); el != nil; el = next {
                next = el.Next()
                elVal := el.Value.(Subscription)
                if elVal.Evaluate(msg, i) && elVal.GetOnce() {
                    subscriptions.Remove(el)
                }
            }
        }
    }
}
func (i *IRC) Quit() {
    log.Print("Quit")
    close(i.quit)
}
