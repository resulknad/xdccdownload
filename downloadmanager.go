package main
import "fmt"
import "sync"
import "path"


type Download struct {
    Packid int
    Targetfolder string
    Pack Package
    Percentage float32
    Messages string
}

func CreateDownloadManager(i *Indexer, c *Config, connPool *ConnectionPool) *DownloadManager{
    q := make(chan bool)
    return &DownloadManager{Indx:i, quit: q, lock: sync.Mutex{}, Conf: c, connPool: connPool}
}

type DownloadManager struct {
    List []*Download
    Indx *Indexer
    Conf *Config
    lock sync.Mutex
    connPool *ConnectionPool
    quit chan bool
}

func (dm *DownloadManager) GetDownload(id int) (int,*Download) {
    dm.lock.Lock()
    defer dm.lock.Unlock()
    for indx,el:= range dm.List {
        if el.Pack.ID == uint(id) {
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

func (dm *DownloadManager) DeleteOne(id int) {
    dm.lock.Lock()
    defer dm.lock.Unlock()
    found, _ := dm.GetDownload(id)
    if found >-1 {
        dm.List =append(dm.List[:found],dm.List[found+1:]...)
    }
}

func (dm *DownloadManager) DoDownload(d *Download) {
    p := (*d).Pack
    i := dm.connPool.GetConnection(p.Server)
    if i == nil {
        dm.lock.Lock()
        d.Percentage = -1
        dm.lock.Unlock()
        return
    }
    ch := make(chan XDCCDownloadMessage, 200)
    x := XDCC{Bot: p.Bot, Channel: p.Channel, Package: p.Package, IRCConn: i, Filename: p.Filename}
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
            }
            dm.lock.Unlock()
        }
    }
    u := Unpack{dm.Conf.TempPath, path.Join(dm.Conf.TargetPath, d.Targetfolder), filePath}
    u.Do()
}

func printSlice(s []*Download) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}

func (dm *DownloadManager) CreateDownload(d Download) {
    dm.lock.Lock()
    defer dm.lock.Unlock()
    printSlice(dm.List)
    d.Percentage = 0.0
    dm.List = append(dm.List, &d)
    printSlice(dm.List)
    go dm.DoDownload(&d)
}
