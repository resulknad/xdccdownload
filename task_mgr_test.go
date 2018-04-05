package main

import "testing"
import "os"
import "time"

func CTaskmgr() *Taskmgr {
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

	return CreateTaskmgr(indx, dm)
}

func TestMgrCRUD(t *testing.T) {
	tm := CTaskmgr()	
	ti := &Taskinfo{Name: "Test1", Criteria: ".SizeMbytes > 0 && .SizeMbytes < 100 && .Filename contains Homeland"}
	tm.CreateTask(ti)
	time.Sleep(10*time.Second)
	ti.Name = "Test2"
	tm.UpdateTask(ti)
	time.Sleep(10*time.Second)
	// todo test smth
}

