package main

import "sync"
import "log"
import "fmt"
import (
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
)
type Taskmgr struct {
	sync.Mutex
	indx *Indexer
	dlm *DownloadManager
	db *gorm.DB
	tasks []*Task
}

func CreateTaskmgr(indx *Indexer, dlm *DownloadManager) *Taskmgr {
	t := Taskmgr{}
	t.indx = indx
	t.db = indx.db

	t.db.AutoMigrate(&Taskinfo{})
	t.dlm = dlm
	return &t
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
	go t.EnqueueAllFromDB(true)
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
