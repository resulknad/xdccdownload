package main
import "log"
import "time"

import (
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
	quit chan bool
	indx *Indexer
	dlm *DownloadManager
}



func (t* Task) Init(indx *Indexer, dlm *DownloadManager) {
	t.State = 1
	t.indx = indx
	t.dlm = dlm
	t.queue = make(chan Package, 10)
	t.quit = make(chan bool)
}

func (t *Task) MatchesCriterias(p Package) bool {
	m := structs.Map(p)
	m["SizeMbytes"] = p.SizeMbytes()
	for k,v := range structs.Map(t.indx.getReleaseForPackage(p).Release) {
		m["r" + k] = v
	}
	match, err := ParseAndEval(t.Criteria, m)
	if err != nil{
		log.Print("Error evaluating " + t.Criteria)
		return false
	}
	//log.Print(p)
	return match
}

func (t *Task) EnqueueAllFromDB(block bool) {
    var pckgs []Package
    t.indx.db.Find(&pckgs)
	for _, p := range(pckgs) {
		if t.MatchesCriterias(p) && !t.indx.CheckDownloaded(p) {
			if (!t.enqueue(p,block)) {
				return 
			}
			if t.CheckQuit() {
				log.Print("quitting enqueueall")
				return
			}
			log.Print(p)
			log.Print("added one")
		}
	}
}

func (t *Task) enqueue(p Package, block bool) bool {
	if block {
		t.queue<-p
		return true
	}
	select {
	case t.queue<-p:
		return true
	default:
		return false
	}
}

func (t *Task) CheckQuit() bool {
	select {
	case <-t.quit:
		return true
	default:
		return false
	}
}

func (t *Task) Worker() {
	
	t.State = 2
	for {
		select {
		case p:=<-t.queue:
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
		case <-t.quit:
			t.State = 0
			log.Print("Quit task")
			return
		}
	}
}
