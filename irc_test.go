package main

import "testing"

func TestCheckChannels(t *testing.T) {
	irc := IRC{Server: "irc.abjects.net:6667"}
	if !irc.Connect() {
		t.FailNow()
	}
	irc.JoinChannel("#sports-bar")
	if !irc.CheckChannel("#sports-bar") {
		t.FailNow()
	}
	irc.CommandCh<-"PART #sports-bar\r\n"
	if irc.CheckChannel("#sports-bar") {
		t.FailNow()
	}
	irc.channels = []string{}
	irc.JoinChannel("#sports-bar")
	if !irc.CheckChannel("#sports-bar") {
		t.FailNow()
	}
}
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
