
package main



import "sync"
import "log"
import "fmt"
import "path"
import "encoding/json"
import "strings"
import (
  	"github.com/boltdb/bolt"
)
type Taskmgr struct {
	sync.Mutex
	indx *Indexer
	dlm *DownloadManager
	pckgCh chan PackageMsg
	db *bolt.DB
	tasks []*Task
}
type PackageMsg struct {
	Package Package
	Downloaded bool
}

func CreateTaskmgr(indx *Indexer, dlm *DownloadManager, conf *Config) *Taskmgr {
	t := Taskmgr{}
	t.indx = indx

	p := path.Join(conf.DBPath, ".tasks.bdb")
	db, err := bolt.Open(p, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	db.Update(func(tx *bolt.Tx) error {
	  tx.CreateBucketIfNotExists([]byte("tasks"))
	  return nil
	})
	t.db = db

	t.dlm = dlm

	t.pckgCh = make(chan PackageMsg,100)
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
			if !p.Downloaded {
				tm.EnqueueAll(p.Package)
			}
		}
	}
}

func (tm *Taskmgr) EnqueueAll(p Package) {
	for _,tl := range(tm.tasks) {
		if tl != nil {
			if tl.MatchesCriterias(p) {
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
	t.EmptyQueue()
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

	t.Init(tm.indx, tm.dlm, tm.db)
	tm.indx.TriggerRescan()

	if !t.Taskinfo.Enabled {
		return
	}

	go t.Worker()
}

func (t *Taskmgr) EnqueueAllFromDB() {
	/*for {
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
	}*/
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
	tm.db.Update(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  tBucket.Delete(ti.byteid())
	  return nil
	})
}

func (tm *Taskmgr) UpdateTask(ti *Taskinfo) {
	fmt.Print("Updating")
	tm.QuitTask(tm.titot(ti))
	tm.db.Update(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  tBucket.Put(ti.byteid(), ti.json())
	  return nil
	})
	tm.CreateTask(ti)
}

func (tm *Taskmgr) CreateTask(ti *Taskinfo) {

	if ti.ID == 0 {
	  tm.db.Update(func (tx *bolt.Tx) error {
		tBucket := tx.Bucket([]byte("tasks"))
		nid,_ := tBucket.NextSequence()
		ti.ID = nid
		tBucket.Put(ti.byteid(), ti.json())
		return nil
	  })

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
		ts[len(ts)-1].Packages = ts[len(ts)-1].GetAllQueued()
	}
	return ts
}

func (tm *Taskmgr) GetTask(id uint64) *Task {
	var ti Taskinfo
	tm.db.View(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))
	  ti_json := tBucket.Get(Taskinfo{ID: id}.byteid())
	  if ti_json != nil {
		ti = *TaskinfoFromJSON(ti_json)
	  }
	  
	  return nil
	})

	if t := tm.titot(&ti); t!=nil {
		return t
	} else {
		t := Task{Taskinfo: ti}
		return &t
	}
}

func TaskinfoFromJSON(b []byte) *Taskinfo {
  var ti Taskinfo
  err := json.Unmarshal(b, &ti)
  if err != nil {
	panic(err)
  }
  return &ti
}

func (tm *Taskmgr) GetAllTaskinfo() []*Taskinfo {
	tis := []*Taskinfo{}
	tm.db.View(func (tx *bolt.Tx) error {
	  tBucket := tx.Bucket([]byte("tasks"))

	  c := tBucket.Cursor()
	  for k, v := c.First(); k != nil; k, v = c.Next() {
		if !strings.Contains(string(k), ":") {
		  tis = append(tis, TaskinfoFromJSON(v))
		}
	  }
	  return nil
	})

	return tis
}
