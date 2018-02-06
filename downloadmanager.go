package main
import "fmt"
import "sync"
import "path"


type Download struct {
    Packid int
    Targetfolder string
    Pack Package
    Percentage float32
}

func CreateDownloadManager(i *Indexer, c *Config) *DownloadManager{
    q := make(chan bool)
    return &DownloadManager{Indx:i, quit: q, lock: sync.Mutex{}, Conf: c}
}

type DownloadManager struct {
    List []*Download
    Indx *Indexer
    Conf *Config
    lock sync.Mutex
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
    i := IRC{Server: p.Server}
    if i.Connect() == false {
        dm.lock.Lock()
        d.Percentage = -1
        dm.lock.Unlock()
        return
    }
    ch := make(chan float32, 200)
    filenameCh := make(chan string)
    x := XDCC{Bot: p.Bot, Channel: p.Channel, Package: p.Package, IRCConn: &i}
    go x.Download(ch, filenameCh, dm.Conf.TempPath)
    for d.Percentage <1 {
        select {
        case progress := <-ch:
            dm.lock.Lock()
            d.Percentage = progress
            dm.lock.Unlock()
        }
    }
    u := Unpack{dm.Conf.TempPath, path.Join(dm.Conf.TargetPath, d.Targetfolder), <-filenameCh}
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
