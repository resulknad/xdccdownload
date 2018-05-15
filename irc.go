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
import "golang.org/x/net/proxy"
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
	Proxy *proxy.Dialer

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
	GetQuit() *SubscriptionQuit
}

type SubscriptionQuit struct {
	Quit chan bool
}

func (s *SubscriptionQuit) IsClosed() bool {
	if s == nil {
		return false
	}
	select {
		case <-s.Quit:
			return true
		default:
			return false
	}
}
func (s *SubscriptionQuit) Init() {
	s.Quit = make(chan bool)
	log.Print(s)
}
	

type CodeSubscription struct {
    Subscription
	Quit *SubscriptionQuit
    Once bool
    Backchannel chan string
    Code string
}

type PrivMsgSubscription struct {
	Quit *SubscriptionQuit
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
select {
case            f.Backchannel <- PrivMsg{Content: privmsgRegexp.FindStringSubmatch(msg)[2], From: receiverRegexp.FindStringSubmatch(msg)[1], Server: i.Server, Channel: i.Channel, To: privmsgRegexp.FindStringSubmatch(msg)[1]}:
default:
log.Print("privmsg backchannel full " + f.To)
}

        return true
    }
    return false
}
func (f PrivMsgSubscription) GetQuit() *SubscriptionQuit {
    return f.Quit;
}

func (f PrivMsgSubscription) GetOnce() bool {
    return f.Once;
}

type GeneralSubscription struct {
    Subscription
	Quit *SubscriptionQuit
    Once bool
    Backchannel chan string
    Filter string
}
func (f GeneralSubscription) GetQuit() *SubscriptionQuit {
    return f.Quit;
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
	select {
        case f.Backchannel <- msg:
default:
log.Print("general backchannel full " + f.Filter)
}
        return true
    }
    return false
}


func (f CodeSubscription) GetOnce() bool {
    return f.Once;
}
func (f CodeSubscription) GetQuit() *SubscriptionQuit {
    return f.Quit;
}

func (f CodeSubscription) Evaluate(msg string, i *IRC) bool {
	codeRegexp := regexp.MustCompile(`^:[^ ]+ ([0-9]+)`)


    if codeRegexp.MatchString(msg) &&
        codeRegexp.FindStringSubmatch(msg)[1] == f.Code {
        //fmt.Println("rule struck: %s, msg: %s", f.Filter, msg)
select {
case	        f.Backchannel <- removeWholePrefix(msg):
default:
log.Print("code backchannel full " + f.Code)
}
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
func (i *IRC) BanListener(channel chan string){
    for {
        select {
        case ban := <-channel:
			log.Print("Banned from channel..." + ban)
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

    // setup banlistener
    banChan := make(chan string, 10)
    go i.BanListener(banChan)
    i.SubscriptionCh<-CodeSubscription{Once: false, Backchannel: banChan, Code: "474"}
    // conenct to server
	log.Print("Connecting to " + i.Server)

	var conn net.Conn
	var err error
	if i.Proxy != nil {
		conn, err = (*i.Proxy).Dial("tcp", i.Server)
	} else {
		conn, err = net.Dial("tcp", i.Server)
	}
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
func (i *IRC) StillConnected() bool {
    str, _ := gostrgen.RandGen(5, gostrgen.Lower | gostrgen.Upper, "", "")

    pongAwait := make(chan string)
    i.SubscriptionCh<-GeneralSubscription{Filter: "PONG", Backchannel:pongAwait, Once:true}

	i.CommandCh<-("PING "+str+"\r\n")
	select {
	case <-pongAwait:
		return true
	case <-time.After(5*time.Second):
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
		namesSub := CodeSubscription{Once: false, Backchannel: namesCh, Code: "353", Quit: &SubscriptionQuit{}}
		log.Print(namesSub.GetQuit())
		namesSub.GetQuit().Init()
		i.SubscriptionCh<-namesSub

		endNamesCh := make(chan string, 10)
		endNamesSub := CodeSubscription{Once: true, Backchannel: endNamesCh, Code: "363", Quit: &SubscriptionQuit{}}
		endNamesSub.GetQuit().Init()
		i.SubscriptionCh<-endNamesSub

		defer func() {
			log.Print(namesSub)
			close(namesSub.GetQuit().Quit)
			close(endNamesSub.GetQuit().Quit)
		}()

		i.CommandCh<-("NAMES "+c+"\r\n")
		for {
			select {
			case list := <-namesCh:
				if strings.Contains(list, i.Nick) {
					log.Print("good")
					//log.Print(list)
					return true
				}
			case <-endNamesCh:
				log.Print("end names not good")
				i.removeChannel(c)
				return false
	
			case <-time.After(10*time.Second):
				log.Print("timeout not good")
				i.removeChannel(c)
				return false
			}
		}

		log.Print("good")
		return true
}
func (i *IRC) removeChannel(channel string) { // unlocked! must be called within lock
	for j,v := range(i.channels) {
		if channel == v {
			i.channels[j] = i.channels[len(i.channels)-1] // Replace it with the last one.
			i.channels = i.channels[:len(i.channels)-1]   // Chop off the last one.
		}

	}
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
                if elVal.GetQuit().IsClosed() || (elVal.Evaluate(msg, i) && elVal.GetOnce()) {
                    subscriptions.Remove(el)
                }
            }
        }
    }
}
func (i *IRC) Quit() {
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered, couldnt close irc channel, possibly closed before")
		}
	}()
    log.Print("Quit")
    close(i.quit)
}
