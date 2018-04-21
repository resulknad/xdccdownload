package main
import "log"

import (
	"time"
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
	 "github.com/fatih/structs"
)

type Taskinfo struct {
	gorm.Model
	Name string
	Criteria string
	Enabled bool
}

type Task struct {
	Taskinfo
	State int
	queue chan Package
	parsedExp *ParsedExpression
	quit chan bool
	indx *Indexer
	dlm *DownloadManager
}


type TaskQueue struct {
	gorm.Model
	TaskinfoID uint
	PackageID uint
	Package Package `gorm:"foreignkey:PackageID"`
}

func (t* Task) Init(indx *Indexer, dlm *DownloadManager) {
	t.State = 1
	t.indx = indx
	t.dlm = dlm
	err,pe := ParseToPE(t.Criteria)
	
	if err != nil || t.Criteria == "" {
	} else {
		t.parsedExp = pe
	}
	t.queue = make(chan Package, 10)
	t.quit = make(chan bool)
}

func (t *Task) MatchesCriterias(p Package) bool {
	m := structs.Map(p)
	m["SizeMbytes"] = p.SizeMbytes()
	for k,v := range structs.Map(t.indx.getReleaseForPackage(p).Release) {
		m["r" + k] = v
	}
	for k,v := range structs.Map(t.indx.getReleaseForPackage(p)) {
		m["r" + k] = v
	}
	if t.parsedExp == nil {
		return false
	}
	err, match := t.parsedExp.Eval(m)
	if err != nil{
		log.Print("Error evaluating " + t.Criteria)
		return false
	}
	//log.Print(p)
	return match
}



func (t *Task) enqueue(p Package, block bool) bool {
	if ! (p.ID > 0) {
		return false
	}

	t.indx.db.Create(&TaskQueue{PackageID:p.ID,TaskinfoID:t.ID})
	return true
}

func (t *Task) CheckQuit() bool {
	select {
		case <-t.quit:
			return true
		default:
			return false
	}
}

func (t *Task) PullFromQueue() (bool, *TaskQueue) {
	var q TaskQueue
	tx := t.indx.db.Begin()
	tx.Preload("Package").Where("taskinfo_id = ?", t.ID).First(&q)
	if q.ID == 0 {
		tx.Commit()
		return false, nil
	}

	tx.Delete(&q)
	tx.Commit()

	if q.Package.Filename == "" { // package might have been deleted at some point
		return false, nil
	}
	return true, &q
}

func (t *Task) Worker() {
	
	t.State = 2
	for {
		if t.CheckQuit() {
			t.State = 0
			log.Print("Quitting task")
			return
		}

		avail, q := t.PullFromQueue()

		if !avail {
			time.Sleep(1*time.Second)
			continue
		}
		p := q.Package

		if t.indx.CheckDownloaded(p) {
			continue
		}
		dlId := t.dlm.CreateDownload(Download{Pack: p, Targetfolder:p.TargetFolder()})
		i, dl := t.dlm.GetDownload(dlId)
		log.Print("Issued DL ")

		for i != -1 && dl.State == 0 { // wait while DL in progress
			time.Sleep(1*time.Second)
			i, dl = t.dlm.GetDownload(dlId)
			log.Print("loop")
			if t.CheckQuit() {
				t.State = 0
				log.Print("Quitting task")
				return
			}
		}
		if dl.State == -1 {
			t.indx.RemovePackage(&p)
			log.Print("Deleted package bc of failure to dl")
		} else {
			t.indx.AddDownloaded(p)
		}
		log.Print("Done")


	}
}
