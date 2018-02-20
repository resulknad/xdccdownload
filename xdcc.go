package main

import "net/url"
import "path"
import "net"
import "fmt"
import "time"

//import "strings"
import "regexp"
import "encoding/binary"
import "os"
import "strconv"

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
    fmt.Println("Nick: %s", i.IRCConn.Nick)
	i.IRCConn.SubscriptionCh <- PrivMsgSubscription{To: i.IRCConn.Nick, Backchannel: awaitFeedback, Once: true}

	var feedback string
	recv := false

	feedback, recv = i.awaitFeedbackAfterRequest(awaitFeedback)
    a := 0
	for a = 0; a < 10 && !OfferMatchesDesired(feedback); a++ {
        prog<- XDCCDownloadMessage{Message: "Try: " + strconv.Itoa(a)}
		i.IRCConn.CommandCh <- fmt.Sprintf("PRIVMSG %s :xdcc send %s", i.Bot, i.Package)
		feedback, recv = i.awaitFeedbackAfterRequest(awaitFeedback)
        if recv {
            prog<- XDCCDownloadMessage{Message: feedback}
        }

	}

	if (a >=10) {
        prog<- XDCCDownloadMessage{Err: "No dcc send from bot"}
		i.IRCConn.CommandCh <- fmt.Sprintf("PRIVMSG %s :xdcc remove", i.Bot) // we might be on some queue...
		return false
	}

	fmt.Print("got feedback")
	fmt.Print(feedback)


    offer := i.ParseSend(feedback)

	conn, err := net.Dial("tcp", offer.IP+":"+offer.Port)

	if err != nil {
        prog<- XDCCDownloadMessage{Err: string(err.Error())}
		return false
	}
	var recvBytes int64
    var recvBytesSinceLastAck int64
	recvBytes = 0
    recvBytesSinceLastAck = 0
	recvBuf := make([]byte, 4096)
    pathToFile := path.Join(tempdir, url.PathEscape(offer.Filename))
	f, err := os.Create(pathToFile)
	defer f.Close()
	timeLastRecv := time.Now()
	var samplingN int64
	G:
	for {
		n, err2 := conn.Read(recvBuf[:]) // recv data
		if err2 != nil {
            // we will try to recover from those like below
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("read timeout:", netErr, n)
			} else {
				fmt.Println("read error:", err2, n)
			}

            // as implemented in HexChat
            if recvBytesSinceLastAck != 0 && recvBytes != offer.Size && recvBytes <= (1<<31 - 1) {
                bs := make([]byte, 4)
                binary.BigEndian.PutUint32(bs, uint32(recvBytesSinceLastAck))
                recvBytesSinceLastAck = 0
                conn.Write(bs)
            } else { // will only enter the above case once, second time we break
                break G
            }
		}
		if (samplingN > int64(i.Conf.SpeedLimit)*int64(1024)) {
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
		recvBytes = recvBytes + int64(n)
        recvBytesSinceLastAck += int64(n)
		f.Write(recvBuf[:n])

        prog<-XDCCDownloadMessage{Progress: float32(recvBytes)/float32(offer.Size)}
		if recvBytes == (offer.Size) {
			fmt.Println("Received file.")
			break G
		}

	}
    f.Sync()
	f.Close()
    prog<-XDCCDownloadMessage{Filename: pathToFile}
	return  (recvBytes) == (offer.Size)
}

