package main


import "log"

import (
	"time"
	"bytes"
	"strconv"
	"encoding/json"
  	"github.com/boltdb/bolt"
	 "github.com/fatih/structs"
)

type Taskinfo struct {
  ID uint64
  Name string
  Criteria string
  Enabled bool
}

func (ti Taskinfo) json() []byte {
  b, err := json.Marshal(ti)
  if err == nil {
	return b
  } else {
	return []byte("")
  }
}
func (ti Taskinfo) byteid() []byte {
  return []byte(strconv.FormatUint(ti.ID,10))
}

type Task struct {
	Taskinfo
	db *bolt.DB
	State int
	queue chan Package
	parsedExp *ParsedExpression
	quit chan bool
	indx *Indexer
	dlm *DownloadManager
	Packages []Package
}


type TaskQueue struct {
  ID string
	TaskinfoID uint
	PackageID uint
	Package Package `gorm:"foreignkey:PackageID"`
}

func (t* Task) Init(indx *Indexer, dlm *DownloadManager, db *bolt.DB) {
  t.db = db
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
  if len(t.GetAllQueued()) > 10 {
	log.Print("queue full, ignore")
	return false
  }
	t.db.Update(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  c := tBucket.Cursor()
	  prefix := append(t.Taskinfo.byteid(), []byte(":")...)

	  for k,v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
		if PackageFromJSON(v).key() == p.key() {
		  log.Print("didnt add bc already in there")
		  return nil
		}
	  }

	  nid, _ := tBucket.NextSequence()
	  tBucket.Put([]byte(strconv.FormatUint(t.ID, 10) + ":" + strconv.FormatUint(nid, 10)), []byte(p.json()))
	  return nil
	})

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

func (t *Task) GetAllQueued() []Package {
  var res []Package
	t.db.Update(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  c := tBucket.Cursor()
	  prefix := append(t.Taskinfo.byteid(), []byte(":")...)

	  for k,v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
		res = append(res, PackageFromJSON(v))
	  }

	  return nil
	})
	return res
}


func (t *Task) EmptyQueue() {
	t.db.Update(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  c := tBucket.Cursor()
	  prefix := append(t.Taskinfo.byteid(), []byte(":")...)

	  for k,_ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		tBucket.Delete(k)
	  }

	  return nil
	})
}

func (t *Task) PullFromQueue() (found bool, p Package) {
	t.db.Update(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  c := tBucket.Cursor()
	  prefix := append(t.Taskinfo.byteid(), []byte(":")...)

	  k,v := c.Seek(prefix)
	  if k != nil && bytes.HasPrefix(k, prefix) {
		found = true
		p = PackageFromJSON(v)
		tBucket.Delete(k)
	  } else {
		found = false
	  }
	  return nil
	})

	return found, p
}

func (t *Task) Worker() {
	
	t.State = 2
	for {
		if t.CheckQuit() {
			t.State = 0
			log.Print("Quitting task")
			return
		}

		avail, p := t.PullFromQueue()

		if !avail {
			time.Sleep(1*time.Second)
			continue
		}

		if t.indx.CheckDownloaded(p) {
			continue
		}
		dlId := t.dlm.CreateDownload(Download{Pack: p, Targetfolder:p.TargetFolder()})
		i, dl := t.dlm.GetDownload(dlId)
		log.Print("Issued DL ")

		for i != -1 && dl.State == 0 { // wait while DL in progress
			time.Sleep(1*time.Second)
			i, dl = t.dlm.GetDownload(dlId)
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
		  log.Print("add to downloaded")
		  log.Print(p.Release)
			t.indx.AddDownloaded(p.Release)
		}
		log.Print("Done")


	}
}

