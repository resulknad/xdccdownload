package main
/*
import "testing"
import "os"
import "time"
import "fmt"

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

	// indx := CreateIndexer(&c, &connPool)
	// indx.db.Exec("DELETE FROM task_queues; DELETE FROM taskinfos;")
    // dm := CreateDownloadManager(indx, &c, &connPool)

	// return CreateTaskmgr(indx, dm)
	return nil
}

func TestMgrQuit(t *testing.T) {
	tm := CTaskmgr()	
	ti := &Taskinfo{Name: "Test1", Criteria: ".SizeMbytes > 0 && .SizeMbytes < 100 && .Filename contains Homeland", Enabled: true}
	tm.CreateTask(ti)
	time.Sleep(1*time.Second)
	if tm.titot(ti).State != 2 {
		fmt.Print("create failed", tm.titot(ti).State)
		t.FailNow()
	}
	task := tm.titot(ti)
	tm.UpdateTask(ti)
	time.Sleep(3*time.Second)
	if task.State != 0 {
		fmt.Print("quit failed")
		t.FailNow()
	}
	// todo test smth
}

func TestMgrQueue(t *testing.T) {
	tm := CTaskmgr()	
	ti := &Taskinfo{Name: "1", Criteria: "", Enabled: false}
	ti2 := &Taskinfo{Name: "2", Criteria: "", Enabled: false}
	tm.CreateTask(ti)
	tm.CreateTask(ti2)

	t1 := tm.titot(ti)
	t2 := tm.titot(ti2)
	if t1 == t2 {
		t.FailNow()
	}
	

	var i uint
	for i=0; i<100; i++ {
		//t1.enqueue(Package{Model:gorm.Model{ID:i}},false)	
		//t2.enqueue(Package{Model:gorm.Model{ID:100+i}},false)	
	}

	for i:=0; i<100; i++ {
		// suc, tq := t1.PullFromQueue()
		// suc2, tq2 := t2.PullFromQueue()	
		/* if !suc || !suc2 || tq.PackageID > 99 || tq2.PackageID < 100 {
			//fmt.Print(tq,tq2)
			fmt.Print("dequeue error",tq.PackageID,tq2.PackageID)
			t.FailNow()
		}
		*/
	}

	suc, _ := t1.PullFromQueue()
	suc2, _ := t2.PullFromQueue()	
	if suc || suc2 {
		t.FailNow()
	}
}

*/
