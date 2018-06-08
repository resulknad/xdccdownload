package main
import "time"
import "log"
import "path/filepath"
import "encoding/json"
import "strings"
import "strconv"
import "regexp"
import "os"
import "path"
import "fmt"
import (
  	"github.com/boltdb/bolt"
	"github.com/cytec/releaseparser"
)


type Indexer struct {
    Conf *Config
    db *bolt.DB
    connPool *ConnectionPool
	imdb *IMDB
	pckgChs [] (chan Package)
	timeToScan chan bool
	announcementCh chan PrivMsg
}

type Package struct {
	ID string
    Server string
    Channel string
    Bot string
    Package string
    Filename string
    Size string
    Gets string
    Time string
	Release Release
}

type Release struct {
	IMDBId string
	AvgRating float64
	NumVotes int
	releaseparser.Release
}

type Downloaded struct {
	Filename string
	Location string
	ReleaseID uint
}

func (i *Indexer) TriggerRescan() {
  select {
  case i.timeToScan<-true:
  default:
  }
}

func (i *Indexer) OfferAllToTasks() {
  for {
	log.Print("starting to offer all packages to tasks")
	i.db.View(func(tx *bolt.Tx) error {
	  pBucket := tx.Bucket([]byte("packages"))
	  c := pBucket.Cursor()
	  for k, v := c.First(); k != nil; k, v = c.Next() {
		i.offerPackageToTasks(PackageFromJSON(v), true)
	  }
	  return nil
	})
	log.Print("done to offer all packages to tasks")
	select {
	  case <-time.After(10*time.Minute):
	  case <-i.timeToScan:
	  log.Print("full db scan triggered")
	}

  }
}

func (i *Indexer) ExpirePackages() {
  for {
	i.db.Update(func(tx *bolt.Tx) error {
	  expirationTime := time.Now().Add(-4 * time.Hour)
	  pBucket := tx.Bucket([]byte("packages"))
	  c := pBucket.Cursor()
	  for k, v := c.First(); k != nil; k, v = c.Next() {
		p := PackageFromJSON(v)
		t,_ := time.Parse(time.RFC850, p.Time)
		if t.Before(expirationTime) {
		  i.removePackage(tx, &p)
		}
	  }
	  return nil
	})
	

	time.Sleep(10*time.Minute)
  }
}

func (i *Indexer) AddDownloaded(r Release) {
  err := i.db.Update(func(tx *bolt.Tx) error {
	dBucket := tx.Bucket([]byte("downloaded"))
	dBucket.Put([]byte(r.key()), []byte(r.json()))
	log.Print("putting " + r.key() + " = " + string(r.json()))
	return nil
  })
  if err != nil {
	panic(err)
  }
}

func (i *Indexer) CheckDownloadedExact(p Package) bool {
  panic("not implemented")
  return false
}

func (i *Indexer) releaseDownloaded(r *Release) (downloaded bool) {
  err := i.db.Update(func(tx *bolt.Tx) error {
	dBucket := tx.Bucket([]byte("downloaded"))
	if dBucket.Get([]byte(r.key())) != nil {
	  downloaded = true
	} else {
	  downloaded = false
	}
	return nil
  })
  if err != nil {
	panic(err)
  }
  return downloaded
}


func (i *Indexer) ResetDownloaded() bool {
	// tx := i.db.Begin()

	// tx.Exec("DELETE FROM downloadeds;")
	for _,dir := range i.Conf.GetDirs() {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
				return err
			}
			if !info.IsDir() {
			  p := Package{Filename: info.Name()}
			  r := p.Parse()
			  i.AddDownloaded(r)
			}
			fmt.Printf("visited file: %q\n", path)
			return nil
		})

		if err != nil {
		fmt.Printf("error walking the path %q: %v\n", dir, err)
		}

	}
	//err := tx.Commit()
	//return err != nil
	return true
}

func (i *Indexer) CheckDownloaded(p Package) bool {
	//i.db.LogMode(true)
	r := i.getReleaseForPackage(p)//p.Parse() 
	if r.Type == "movie" {
		return i.releaseDownloaded(&Release{Release: releaseparser.Release{Type: r.Type, Title: r.Title, Year: r.Year}})
	} else if (r.Type == "tvshow") {
		return i.releaseDownloaded(&Release{Release: releaseparser.Release{Type: r.Type, Title: r.Title, Season: r.Season, Episode: r.Episode}})
	}
	return false
}

func (i *Indexer) EnrichWithIMDB(r *Release) {
	var id string
	if r.Type == "movie" {
		id = i.imdb.GetIdForMovie(r.Title, r.Year)
	} else {
		id = i.imdb.GetIdForShow(r.Title)
	}
	if id != "" {
		rating, num := i.imdb.GetRating(id)	
		r.IMDBId = id
		r.AvgRating = rating
		r.NumVotes = num
	}
}

func (i *Indexer) EnrichAll() { 
  /*
    var rs []Release
	tx := i.db.Begin()
	i.db.Where(Release{IMDBId:""}).Find(&rs)
	for _, r := range(rs) {
		i.EnrichWithIMDB(&r)
		tx.Save(r)
		log.Print(r)
	}
	tx.Commit()
	*/
}

func (p Package) Parse() Release {
	return Release{Release: *releaseparser.Parse(p.Filename)}
}

func (p *Package) SizeMbytes() float64 {
	size := strings.Trim(p.Size, " ")
	if len(size) == 0 {
		return -1
	}
	lastC := size[len(size)-1:]	
	s,err := strconv.ParseFloat(size[:len(size)-1], 64)
	if err != nil {
		return -1
	}
	if lastC == "M" {
		return s
	}
	if lastC == "G" {
		return s*1024
	}
	return -1
}

func (p *Package) TargetFolder() string {

	// TODO: make sure one cant move outside of download directory...
	pp := p.Parse()
	if pp.Type == "tvshow" {
		return fmt.Sprintf("%s/Season %d/",pp.Title, pp.Season)
	} else {
		return pp.Title
	} 
}

func PackageFromJSON(b []byte) Package {
  var r Package
  err := json.Unmarshal(b, &r)
  if err != nil {
	panic(err)
  }
  return r
}

func ReleaseFromJSON(b []byte) Release {
  var r Release
  err := json.Unmarshal(b, &r)
  if err != nil {
	panic("couldnt parse json")
  }
  return r
}

func (i *Indexer) getReleaseForPackage(p Package) (release Release) {
	if p.Release.Type == "" {
	  err := i.db.View(func(tx *bolt.Tx) error {
		rBucket := tx.Bucket([]byte("releases"))
		if r := rBucket.Get([]byte(p.Filename)); r != nil {
		  release = ReleaseFromJSON(r)
		} else {
		  release = p.Parse()
		}
		return nil
	  })
	  if err != nil {
		panic(err)
	  }
	} else {
	  release = p.Release
	}

	return release
}

func (i *Indexer) AddNewPackageSubscription(ch chan Package) {
	i.pckgChs = append(i.pckgChs, ch)
}

func (i *Indexer) offerPackageToTasks(p Package, block bool) {
  for _,ch := range i.pckgChs {
	if !block {
	  select {
		  case ch<-p:
		  default:
			log.Print("offer channel full")
	  }
	} else {
	  ch<-p
	}
  }
}

func (i *Indexer) AddPackage(p Package) {
  panic("not implemented")
    /*if !i.UpdateIfExists(p) {
	  // p.Release = i.getReleaseForPackage(p)

  }
  */
}

func (p Release) key() string {
  return strings.ToLower(fmt.Sprintf("%s:%d:%s:%d:%d", p.Type, p.Year,p.Title, p.Season, p.Episode))
}

func (p Package) key() string {
  return p.Server + ":" + p.Channel + ":" + p.Bot + ":" + p.Package
}

func (p Package) isDifferentTo(p2 Package) bool {
  if p.Filename != p2.Filename {
	return true
  } else {
	return false
  }
}

func (p Release) json() []byte {
  b, err := json.Marshal(p)
  if err == nil {
	return b
  } else {
	return []byte("")
  }
}
func (p Package) json() []byte {
  b, err := json.Marshal(p)
  if err == nil {
	return b
  } else {
	return []byte("")
  }
}

func (i *Indexer) addPackage(tx *bolt.Tx, p Package) {
	  pBucket := tx.Bucket([]byte("packages"))
	  rBucket := tx.Bucket([]byte("releases"))

	  pDb := pBucket.Get([]byte(p.key()))
	  if pDb == nil || PackageFromJSON(pDb).isDifferentTo(p) {
		i.offerPackageToTasks(p, false)
	  }
	  

	  if release := rBucket.Get([]byte(p.Filename)); release != nil {
		p.Release = ReleaseFromJSON(release)
	  } else {
		p.Release = p.Parse()
		rBucket.Put([]byte(p.Filename), p.Release.json())
	  }
	  p.ID = p.key()
	  
	  pBucket.Put([]byte(p.key()), p.json())

}

/*func (i *Indexer) GetPackage(id int) (bool, Package) {
    var pack Package
    i.db.First(&pack, id)
    return true, pack
}*/

func (i *Indexer) RemovePackage(p *Package) {
  err := i.db.Update(func(tx *bolt.Tx) error {
	i.removePackage(tx, p)
	return nil
  })
  if err != nil {
	panic(err)
  }
}
func (i *Indexer) removePackage(tx *bolt.Tx, p *Package) {
  tx.Bucket([]byte("packages")).Delete([]byte(p.key()))
}

func (i *Indexer) Search(name string) []Package {
  var res []Package
  var count int64
  i.db.View(func(tx *bolt.Tx) error {
	pBucket := tx.Bucket([]byte("packages"))
	words := strings.Split(strings.ToLower(name), " ")

	  c := pBucket.Cursor()
	  for k, v := c.First(); k != nil; k, v = c.Next() {
		  count++
		  containsAll := true
		for _,w := range words {
			if !strings.Contains(strings.ToLower(string(v)), w) {
				containsAll = false
				break
			}
		  }
		  if containsAll {
			  res = append(res, PackageFromJSON(v))
		  }
	  }
	  return nil
  })
  log.Print("searched through ",count)
  return res
}

func (i *Indexer) SetupDB() {
  p := path.Join(i.Conf.DBPath, ".indexer.bdb")

  db, err := bolt.Open(p, 0600, nil)
  if err != nil {
	  log.Fatal(err)
  }
  db.Update(func(tx *bolt.Tx) error {
	tx.CreateBucketIfNotExists([]byte("packages"))
	tx.CreateBucketIfNotExists([]byte("releases"))
	tx.CreateBucketIfNotExists([]byte("downloaded"))
	return nil
  })
  i.db = db
}

func (i *Indexer) GetPackage(id string) (found bool, p Package) {
  i.db.View(func(tx *bolt.Tx) error {
	v := tx.Bucket([]byte("packages")).Get([]byte(id))
	if v == nil {
	  found = false
	} else {
	  found = true
	  p = PackageFromJSON(v)
	}
	return nil
  })
  return found, p
}

func (indx* Indexer) WaitForPackages(ch chan PrivMsg) {
    listingRegexp := regexp.MustCompile(`(#[0-9]*)[^0-9]*([0-9]*x)[^\[]*\[([ 0-9.]+(?:M|G)?)\][^\x21-\x7E]*(.*)`)
	colorRegexp := regexp.MustCompile(`\x03[0-9,]*([^\x03]*)\x03`)
	var cache []Package
    for {
        select {
            case msg := <-ch:
                if listingRegexp.MatchString(msg.Content) {
					matches := listingRegexp.FindStringSubmatch(colorRegexp.ReplaceAllString(msg.Content, "$1"))
					nmb, gets, size, name := matches[1], matches[2], strings.Trim(matches[3]," \r\n\u000f"), strings.Trim(matches[4], " \r\n\u000f")
					cache= append(cache, Package{Server: msg.Server, Channel: msg.To, Bot: msg.From, Package: nmb, Filename: name, Size: size, Gets: gets, Time: time.Now().Format(time.RFC850)})
					if len(cache) > 99 {
						err := indx.db.Update(func(tx *bolt.Tx) error {
							for _,p := range cache {
								indx.addPackage(tx, p)
							}
							return nil
						})

						if err != nil {
						  panic(err)
						}
						log.Print("written to db")
						cache = []Package{}
					}
                }
        }
    }
}

func (indx* Indexer) setupChannelListener(server string, channel string) bool {
	connPool := indx.connPool
	i := connPool.GetConnection(server)
	suc := false
	for a:= 0; a<1&&!suc; a++ {

		suc = /*i.Connect() &&*/ i!=nil && i.JoinChannel(channel)
		time.Sleep(time.Duration(0*a)*time.Second)
		i = connPool.GetConnection(server)
	}

	if !suc {
		log.Print("Coulndt connect to ", server, channel)
		return false
	}

	i.SubscriptionCh<-PrivMsgSubscription{Once:false, Backchannel: indx.announcementCh, To:channel}
	return true
}



func CreateIndexer(c *Config, connPool *ConnectionPool) *Indexer {
  indx := Indexer{Conf: c, connPool: connPool, timeToScan: make(chan bool,1)}
  indx.SetupDB()

  indx.imdb = CreateIMDB(c)

  indx.announcementCh = make(chan PrivMsg, 100)
  for _,el := range c.Channels {
	  if !indx.setupChannelListener(el.Server, el.Channel) {
		  //return nil
	  }
  }

  for i :=0; i<2; i++ {
	  go indx.WaitForPackages(indx.announcementCh)
  }
  go indx.ExpirePackages()
  go indx.OfferAllToTasks()

  return &indx
}

func (indx *Indexer) InitWatchDog() {
	go func() {
		for {
			indx.watchDog()
			time.Sleep(60*time.Second)
		}
	}()
}

func (indx *Indexer) watchDog() bool {
	log.Print("watch dog checking connections")
	connectionReset := false
    for _,el := range indx.Conf.Channels {
		// if CheckChannels fails, we possibly lost connection to the server...
		ircConn := indx.connPool.GetConnection(el.Server)
        if ircConn == nil || !ircConn.StillConnected() {
			log.Print("watch dog resetting " + el.Server)
			indx.connPool.Quit(el.Server)	
			indx.setupChannelListener(el.Server, el.Channel)
			connectionReset = true
        } else if !ircConn.CheckChannel(el.Channel) {
			log.Print("watch dog rejoining " + el.Channel)
			if ircConn.JoinChannel(el.Channel) {
				log.Print("rejoined " + el.Channel)
			} else {
				log.Print("failed to rejoin " + el.Channel)
			}
		}
    }
	return connectionReset
}
