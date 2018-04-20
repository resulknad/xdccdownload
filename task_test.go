package main

import "testing"
import "os"

func CTask() *Task {
	connPool := ConnectionPool{}
	chs := []ChannelConfig{ChannelConfig{Server:"irc.abjects.net:6667", Channel:"#aaaaasdf"}}
	dir := []TargetPath{TargetPath{Type:"movie", Dir:"dlmv"}}
	c := (Config{})
	c.Channels = chs
	c.TempPath = "tmp/"
	c.TargetPaths = dir
	os.Mkdir("dlmv", os.ModePerm)
	os.Mkdir("tmp", os.ModePerm)

	indx := CreateIndexer(&c, &connPool)
    dm := CreateDownloadManager(indx, &c, &connPool)
	t := Task{}
	t.Init(indx, dm)
	return &t
}

func TestEnqueueAll(t *testing.T) {
	return
	// task := CTask()	
	// go task.EnqueueAllFromDB(true)
	// task.Worker()
	// todo test smth
}

