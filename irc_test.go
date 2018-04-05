package main

import "testing"
import "log"
import "time"
import "strconv"


// checks if CheckChannel recognized whether we're in a channel or not
func TestCheckChannels(t *testing.T) {
	irc := IRC{Server: "irc.abjects.net:6667"}
	if !irc.Connect() {
		t.FailNow()
	}
	log.Print("1")
	irc.JoinChannel("#moviegods")
	if !irc.CheckChannel("#moviegods") {
		t.FailNow()
	}
	log.Print("2")
	irc.CommandCh<-"PART #moviegods\r\n"
	time.Sleep(1*time.Second)
	if irc.CheckChannel("#moviegods") {
		t.FailNow()
	}
	log.Print("3")
	irc.channels = []string{}
	irc.JoinChannel("#moviegods")
	if !irc.CheckChannel("#moviegods") {
		t.FailNow()
	}
	log.Print("4")
	irc.Quit()
}

// test if our ping (StillConnected) works
func TestConnCheck(t *testing.T) {
	connPool := ConnectionPool{}
	chs := []ChannelConfig{ChannelConfig{Server:"irc.abjects.net:6667", Channel:"#aaaaasdf"}}
	c := (Config{})
	c.Channels = chs

	CreateIndexer(&c, &connPool)
	irc := connPool.GetConnection(chs[0].Server)
	if !irc.StillConnected() {
		t.FailNow()
	}

	connPool.GetConnection(chs[0].Server).conn.Close()

	if irc.StillConnected() {
		t.FailNow()
	}
}

// checks if the watchdog recognizes loss of connection and reconnects + rejoins
func TestWatchDog(t *testing.T) {
	connPool := ConnectionPool{}
	chs := []ChannelConfig{ChannelConfig{Server:"irc.abjects.net:6667", Channel:"#aaaaasdf"}}
	c := (Config{})
	c.Channels = chs

	indx := CreateIndexer(&c, &connPool)
	connPool.GetConnection(chs[0].Server).conn.Close()
	indx.watchDog()
	if !connPool.GetConnection(chs[0].Server).CheckChannel("#aaaaasdf") {
		t.FailNow()
	}
}

// checks if the watchdog recognizes parting a channel and NOT reconnects but rejoins
func TestWatchdogLong(t *testing.T) {
	connPool := ConnectionPool{}
	chs := []ChannelConfig{ChannelConfig{Server:"irc.abjects.net:6667", Channel:"#a"}}
	for i :=0; i<2; i++ {
		chs = append(chs,ChannelConfig{Server:"irc.abjects.net:6667", Channel:"#a"+strconv.FormatInt(int64(i), 10)})
	}
	c := (Config{})
	c.Channels = chs

	irc := connPool.GetConnection(chs[0].Server)

	indx := CreateIndexer(&c, &connPool)

	irc.CommandCh<-"PART #a1\r\n"
	for i :=0; i<10; i++ {
		if indx.watchDog() {
			t.FailNow()
		}
		time.Sleep(1*time.Second);
	}
}
