package main

import "net/url"
import "path"
import "net"
import "fmt"
import "time"
//import "golang.org/x/net/proxy" 

import "regexp"
import "os"
import "strconv"
import "encoding/binary"
import "log"

type XDCCDownloadMessage struct {
    Progress    float32
    Message     string
    Filename    string
    Err         string
	Speed		int64
}

type XDCC struct {
	IRCConn *IRC
	Conf *Config
	Bot     string
	Channel string
	Package string
    Filename string
	Quit	chan bool
}

func (i *XDCC) awaitFeedbackAfterRequest(ch chan PrivMsg) (string, bool) {
	select {
	case feedback := <-ch:
		return feedback.Content, true
	case <-time.After(15 * time.Second):
		return "", false
	}
}

type SendReq struct {
    Filename string
    IP string
    Port string
    Size int64
}

func (i *XDCC) ParseSend(feedback string) *SendReq {
	r := regexp.MustCompile(`DCC SEND ((?:"[^"]+")|(?:[^ ]+)) ([0-9]*) ([0-9]*) ([0-9]*)`)
    if !r.MatchString(feedback) {
        return nil
    }
	s := r.FindStringSubmatch(feedback)
	filename, ip, port, size := s[1], s[2], s[3], s[4]
	fmt.Printf("File: %s, ip: %s, port: %s, size: %s", filename, ip, port, size)
	a, _ := strconv.ParseInt(ip, 10, 64)
	fmt.Println(a)
	ipstr := fmt.Sprintf("%d.%d.%d.%d", byte(a>>24), byte(a>>16), byte(a>>8), byte(a))

    var sizeI int64
	sizeI, _ = strconv.ParseInt(size, 10, 64)
    return &SendReq{Filename: filename, IP:ipstr, Port: port, Size: sizeI}
}

func (i *XDCC) quitSignalRecv(prog chan XDCCDownloadMessage) bool {
	select {
	case <-i.Quit:
		prog<-XDCCDownloadMessage{Err: "quit"}
		return true
	default:
		return false
	}
}

func (i *XDCC) Download(prog chan XDCCDownloadMessage, tempdir string) bool {
    OfferMatchesDesired := func (offer string) bool {
        parsed := i.ParseSend(offer)
        if parsed == nil {
            return false
        }
        fmt.Println("got a dcc send")
        return (parsed.Filename == i.Filename)
    }
	if (!i.IRCConn.JoinChannel(i.Channel)) {
        prog<- XDCCDownloadMessage{Err: "Joining channel failed"}
        return false
    }
	awaitFeedback := make(chan PrivMsg)
    fmt.Println("Nick: ", i.IRCConn.Nick)
	i.IRCConn.SubscriptionCh <- PrivMsgSubscription{To: i.IRCConn.Nick, Backchannel: awaitFeedback, Once: true}

	var feedback string
	recv := false
	CMAXTRIES := 2
	feedback, recv = i.awaitFeedbackAfterRequest(awaitFeedback)
    a := 0
	for a = 0; a < CMAXTRIES && !OfferMatchesDesired(feedback); a++ {
		
        prog<- XDCCDownloadMessage{Message: "Try: " + strconv.Itoa(a)}
		i.IRCConn.CommandCh <- fmt.Sprintf("PRIVMSG %s :xdcc send %s", i.Bot, i.Package)
		feedback, recv = i.awaitFeedbackAfterRequest(awaitFeedback)
        if recv {
            prog<- XDCCDownloadMessage{Message: feedback}
        }
		if i.quitSignalRecv(prog) {
			return false
		}
	}

	if (a >=CMAXTRIES) {
        prog<- XDCCDownloadMessage{Err: "No dcc send from bot"}
		i.IRCConn.CommandCh <- fmt.Sprintf("PRIVMSG %s :xdcc remove", i.Bot) // we might be on some queue...
        	prog<- XDCCDownloadMessage{Err: "no dcc send"}
		return false
	}

	fmt.Print("got feedback")
	fmt.Print(feedback)


    	offer := i.ParseSend(feedback)

	var conn net.Conn
	var err error
	if proxyDial := i.Conf.GetProxyDial(); proxyDial != nil {
		conn, err = (*proxyDial).Dial("tcp", offer.IP+":"+offer.Port)
	} else {
		conn, err = net.Dial("tcp", offer.IP+":"+offer.Port)
	}

	if err != nil {
        prog<- XDCCDownloadMessage{Err: string(err.Error())}
		return false
	}
	var recvBytes uint64
	recvBytes = 0
	
	mirc64 := false
	if offer.Size>>32 > 0 {
		mirc64 = true	
		log.Print("Mirc64")
	} else {
		log.Print("Mirc32")
	}

	recoveredErr := false

	recvBuf := make([]byte, 4096)
    pathToFile := path.Join(tempdir, url.PathEscape(offer.Filename))
	f, err := os.Create(pathToFile)
	defer f.Close()
	timeLastRecv := time.Now()
	var samplingN int64
	G:
	for {
		conn.SetReadDeadline(time.Now().Add(20*time.Second))
		if int64(recvBytes) == (offer.Size) { // maybe EOF => done
			fmt.Println("Received file.")
			break G
		}
		n, err2 := conn.Read(recvBuf[:]) // recv data
		if err2 != nil {
			// try recovering, some bots need this...
			if (mirc64) {
				bs := make([]byte, 8)
				binary.BigEndian.PutUint64(bs, uint64(recvBytes))
				conn.Write(bs)

			} else {
				bs := make([]byte, 4)
				binary.BigEndian.PutUint32(bs, uint32(recvBytes))
				conn.Write(bs)
			}
			log.Print("sending recv bytes...")

			if (recoveredErr) { // if this didnt work, we exit
				prog<-XDCCDownloadMessage{Err: err2.Error()}
				return false
			}

			recoveredErr = true
		} else {
			recoveredErr = false // we weither recovered or the error never happened
		}

		if (i.Conf.SpeedLimit != 0 && samplingN > int64(i.Conf.SpeedLimit)*int64(1024)) {
			elapsed := time.Since(timeLastRecv)
			if elapsed < time.Duration(time.Second) {
				time.Sleep(time.Duration(time.Second)-elapsed)
			}
			elapsed = time.Since(timeLastRecv)
			timeLastRecv = time.Now()
			speed := int64(float64(samplingN)/elapsed.Seconds()/1024)
			prog<-XDCCDownloadMessage{Speed: speed}
			samplingN = 0
		} else {
			samplingN += int64(n)
		}
		
		recvBytes = recvBytes + uint64(n)
		//recvBytesSinceLastAck += int64(n)
		// send ack



		f.Write(recvBuf[:n])

        prog<-XDCCDownloadMessage{Progress: float32(recvBytes)/float32(offer.Size)}

		if i.quitSignalRecv(prog) {
			return false
			// todo: cleanup
		}
	}
    f.Sync()
	f.Close()
    prog<-XDCCDownloadMessage{Filename: pathToFile}
	return  int64(recvBytes) == (offer.Size)
}
