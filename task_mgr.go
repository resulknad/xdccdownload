
package main
/*


import "sync"
import "log"
import "fmt"
import "time"
import (
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
)
type Taskmgr struct {
	sync.Mutex
	indx *Indexer
	dlm *DownloadManager
	pckgCh chan Package
	db *gorm.DB
	tasks []*Task
}

func CreateTaskmgr(indx *Indexer, dlm *DownloadManager) *Taskmgr {
	t := Taskmgr{}
	t.indx = indx
	t.db = indx.db


	t.db.AutoMigrate(&Taskinfo{})
	t.db.AutoMigrate(&TaskQueue{})
	t.dlm = dlm

	t.pckgCh = make(chan Package,100)
	indx.AddNewPackageSubscription(t.pckgCh)
	go t.PackageWorker()
	//go t.EnqueueAllFromDB()
	return &t
}

func (tm *Taskmgr) PackageWorker() {
	// newly added packages
	for {
		select {
		case p:=<-tm.pckgCh:
			tm.EnqueueAll(p)
		}
	}
}

func (tm *Taskmgr) EnqueueAll(p Package) {
	for _,tl := range(tm.tasks) {
		if tl != nil {
			if tl.MatchesCriterias(p) && !tm.indx.CheckDownloaded(p) {
				go tl.enqueue(p, true)
			}
		}
	}
}
func (tm *Taskmgr) QuitTask(t *Task) {
	defer func() {
		log.Print("QuitTask")
		if r := recover(); r!=nil {
			log.Print("recovered", r)
		}
	}()
	log.Print(t)
	close(t.quit)
	for i,tl := range(tm.tasks) {
		if t == tl {
			tm.tasks[i] = nil
		}
	}
}

func (tm *Taskmgr) StartAllTasks() {
	for _,t := range(tm.GetAllTaskinfo()) {
		tm.CreateTask(t)
	}
}

func (tm *Taskmgr) StartTask(t *Task) {

	if t.State > 0 {
		log.Print("quitting in start task, bc task state >0")
		tm.QuitTask(t)
	}

	t.Init(tm.indx, tm.dlm)

	if !t.Taskinfo.Enabled {
		return
	}

	go t.Worker()
}

func (t *Taskmgr) EnqueueAllFromDB() {
	for {
		var pckgs []Package
		// dirty trick, gorm cant preload if we dont page
		for i:=0; (len(pckgs)>0 || i==0); i+=500 {
			log.Print("massive query")
			t.indx.db.Preload("Release").Offset(i).Limit(500).Find(&pckgs)
			for _, p := range(pckgs) {
				t.EnqueueAll(p)
			}
			time.Sleep(5*time.Second) // this process shouldnt put too much load on the system...
		}
		time.Sleep(60*time.Second) // this process shouldnt put too much load on the system...
		
		log.Print("done with enqueue")
	}
}

func (tm *Taskmgr) titot(ti *Taskinfo) *Task {
	for _,t := range(tm.tasks) {
		if t!=nil && t.ID == ti.ID {
			return t
		}
	}
	return nil
}

func (tm *Taskmgr) RemoveTask(ti *Taskinfo) {
	tm.QuitTask(tm.titot(ti))
	tm.db.Delete(&ti)		
}

func (tm *Taskmgr) UpdateTask(ti *Taskinfo) {
	fmt.Print("Updating")
	tm.QuitTask(tm.titot(ti))
	tm.db.Save(ti)
	tm.CreateTask(ti)
}

func (tm *Taskmgr) CreateTask(ti *Taskinfo) {
	if ti.ID == 0 {
		tm.db.Create(ti)	
	}
	t := Task{Taskinfo: *ti}
	tm.tasks = append(tm.tasks, &t)
	tm.StartTask(&t)
}

func (tm *Taskmgr) GetAllTasks() []*Task {
	ts := []*Task{}
	for _,ti := range(tm.GetAllTaskinfo()) {
		if t := tm.titot(ti); t!=nil {
			ts = append(ts, t)
		} else {
			t := Task{Taskinfo: *ti}
			ts = append(ts, &t)
		}
		
	}
	return ts
}


func (tm *Taskmgr) GetTask(id int) *Task {
	var ti Taskinfo
	tm.db.First(&ti, id)
	if t := tm.titot(&ti); t!=nil {
		return t
	} else {
		t := Task{Taskinfo: ti}
		return &t
	}
}

func (tm *Taskmgr) GetAllTaskinfo() []*Taskinfo {
	tis := []*Taskinfo{}
	tm.db.Find(&tis)

	return tis
}
*/
