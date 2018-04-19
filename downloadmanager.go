package main
import "fmt"
import "sync"
import "path"
import "os"
import "time"
import "github.com/elgs/gostrgen"



type Download struct {
	ID string
    Packid int
    Targetfolder string
    Pack Package
    Percentage float32
	Speed int64
    Messages string
	State int
	Quit chan bool `json:"-"`
}

func CreateDownloadManager(i *Indexer, c *Config, connPool *ConnectionPool) *DownloadManager{
    q := make(chan bool)
	dm := &DownloadManager{Indx:i, quit: q, lock: sync.Mutex{}, Conf: c, connPool: connPool}
	dm.downloadCh = make(chan *Download, 100)
    go dm.DownloadWorker()
	return dm
}

type DownloadManager struct {
    List []*Download
    Indx *Indexer
    Conf *Config
    lock sync.Mutex
    connPool *ConnectionPool
    quit chan bool
	downloadCh chan *Download
}

func (dm *DownloadManager) GetDownload(id string) (int,*Download) { // locked fn for outside access
    dm.lock.Lock()
    defer dm.lock.Unlock()
	return dm.getDownload(id)
}

func (dm *DownloadManager) getDownload(id string) (int,*Download) {
    for indx,el:= range dm.List {
        if el.ID == (id) {
            return indx, el
        }
    }
    return -1,nil
}

func (dm *DownloadManager) AllDownloads() []Download {
    dm.lock.Lock()
    defer dm.lock.Unlock()

    // initialize
    dls := make([]Download, 0)

    // create copy, dereference dl objs
    for _,el:= range dm.List {
        fmt.Println(el)
        dls = append(dls, *el)
    }

    return dls
}

func (dm *DownloadManager) DeleteOne(id string) {
    dm.lock.Lock()
    defer dm.lock.Unlock()
    found, dl := dm.getDownload(id)
    if found >-1 {
		dl.Quit<-true
        //dm.List =append(dm.List[:found],dm.List[found+1:]...)
    }
}

func (dm *DownloadManager) DownloadWorker() {
	F:
	for {
		select {
		case d := <-dm.downloadCh:
			dm.lock.Lock()
			d.Messages = "started"
			dm.lock.Unlock()
		p := (*d).Pack
		i := dm.connPool.GetConnection(p.Server)
		if i == nil {
			dm.lock.Lock()
			d.Messages = "couldnt connect"
			dm.lock.Unlock()
			continue F
		}

		ch := make(chan XDCCDownloadMessage, 200)
		if os.Getenv("FAKEDL") == "1" {
			time.Sleep(3*time.Second)
			d.Messages = "Skipped dl for debugging"
			d.State = 1
			continue
		}

			
		x := XDCC{Bot: p.Bot, Channel: p.Channel, Package: p.Package, IRCConn: i, Filename: p.Filename, Conf: dm.Conf, Quit: d.Quit}
		go x.Download(ch, dm.Conf.TempPath)
		var filePath string
		for filePath == "" {
			select {
			case msg := <-ch:
				dm.lock.Lock()
				if msg.Progress != 0 {
					d.Percentage = msg.Progress
				} else if msg.Message != "" {
					d.Messages += msg.Message + "\n"
				} else if msg.Filename != "" {
					filePath = msg.Filename
				} else if msg.Err != "" {
					d.Messages += msg.Err + "\n"
					d.State = -1
					dm.lock.Unlock()
					continue F
				} else if msg.Speed != 0 {
					d.Speed = msg.Speed
				}
				dm.lock.Unlock()
			}
		}
		d.Messages += "Unpacking..."
		u := Unpack{dm.Conf.TempPath, path.Join(dm.Conf.GetTargetDir(p.Parse().Type), d.Targetfolder), filePath}
		u.Do()
		d.State = 1
		d.Messages += "done"
	}
}
}

func printSlice(s []*Download) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}

func (dm *DownloadManager) CreateDownload(d Download) string {
    dm.lock.Lock()
    defer dm.lock.Unlock()
	d.ID,_ = gostrgen.RandGen(15, gostrgen.Lower | gostrgen.Upper, "", "")
	d.State = 0
    d.Percentage = 0.0
	d.Quit = make(chan bool, 10)
    dm.List = append(dm.List, &d)
    dm.downloadCh<-&d
	return d.ID
}
