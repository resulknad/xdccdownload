package main
import "time"
import "log"
import "path/filepath"
import "strings"
import "strconv"
import "regexp"
import "os"
import "path"
import "fmt"
import (
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
	 "github.com/cytec/releaseparser"
)


type Indexer struct {
    Conf *Config
    db *gorm.DB
    connPool *ConnectionPool
	imdb *IMDB
	pckgChs [] (chan Package)
	announcementCh chan PrivMsg
}

type Package struct {
    gorm.Model
    Server string `gorm:"index:pckg"`
    Channel string `gorm:"index:pckg"`
    Bot string `gorm:"index:pckg"`
    Package string `gorm:"index:pckg"`
    Filename string
    Size string
    Gets string
    Time string
	ReleaseID uint
	Release Release  `gorm:"foreignkey:ReleaseID"` 
}

type Release struct {
	IMDBId string
	AvgRating float64
	NumVotes int
	releaseparser.Release
	gorm.Model
}

type Downloaded struct {
	gorm.Model
	Filename string
	Location string
	ReleaseID uint
}

func (i *Indexer) AddDownloaded(p Package) {
	r := i.getReleaseForPackage(p)
	rID := r.ID

	d := Downloaded{Filename: p.Filename, Location: "", ReleaseID: rID}
    i.db.Create(&d)
}

func (i *Indexer) CheckDownloadedExact(p Package) bool {
	//i.db.LogMode(true)
	r := i.getReleaseForPackage(p)//p.Parse() 
	r.ID =	0 
	return i.releaseDownloaded(&r)
}

func (i *Indexer) releaseDownloaded(r *Release) bool {
		res, err := i.db.Table("releases").Select("*").Joins("left join downloadeds on releases.id = downloadeds.release_id").Where(r).Where("downloadeds.id > -1").Limit(1).Rows()	
		defer res.Close()
		return (err == nil) && (res.Next())

}

func (i *Indexer) ResetDownloaded() bool {
	tx := i.db.Begin()

	tx.Exec("DELETE FROM downloadeds;")
	for _,dir := range i.Conf.GetDirs() {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
				return err
			}
			if !info.IsDir() {
				p := Package{Filename: info.Name()}
				r := i.getReleaseForPackage(p)
				rID := r.ID
				d := Downloaded{Filename: p.Filename, Location: "", ReleaseID: rID}
				tx.Create(&d)
			}
			fmt.Printf("visited file: %q\n", path)
			return nil
		})

		if err != nil {
		fmt.Printf("error walking the path %q: %v\n", dir, err)
		}

	}
	err := tx.Commit()
	return err != nil

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
    var rs []Release
	tx := i.db.Begin()
	i.db.Where(Release{IMDBId:""}).Find(&rs)
	for _, r := range(rs) {
		i.EnrichWithIMDB(&r)
		tx.Save(r)
		log.Print(r)
	}
	tx.Commit()
}

func (p *Package) Parse() releaseparser.Release {
	return *releaseparser.Parse(p.Filename)
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

func (i *Indexer) getReleaseForPackage(p Package) Release {
	if p.Release.ID != 0 {
		return p.Release
	}
	var release Release
	for release.ID == 0 {
		parsed := p.Parse()	
		i.db.Where(&parsed).First(&release)
		if release.ID == 0 {
			release.Release = parsed
			i.EnrichWithIMDB(&release)
			i.db.Save(&release)
		}
	}
	return release	
}

func (i *Indexer) AddNewPackageSubscription(ch chan Package) {
	//i.pckgChs = append(i.pckgChs, ch)
}

func (i *Indexer) AddPackage(p Package) {
	p.ReleaseID = i.getReleaseForPackage(p).ID
    if !i.UpdateIfExists(p) {
        i.db.Create(&p)
		for _,ch := range i.pckgChs {
			ch<-p
		}
    }
}

func (i *Indexer) UpdateIfExists(p Package) bool {
    var pDb Package
    i.db.Where("Server = ? AND Bot=? AND Package=? AND Channel=?", p.Server, p.Bot, p.Package, p.Channel).First(&pDb)
    if pDb.ID > 0  { // gorms wants this
        if pDb.Filename != p.Filename {
            i.db.Model(&pDb).Updates(Package{Filename: p.Filename, Size: p.Size, Gets: p.Gets, Time: time.Now().Format(time.RFC850)})
        }
        return true
    }
    return false
}

func (i *Indexer) GetPackage(id int) (bool, Package) {
    var pack Package
    i.db.First(&pack, id)
    return true, pack
}
func (i *Indexer) RemovePackage(p *Package) {
    i.db.Delete(p) // cleanup
}

func (i *Indexer) Search(name string) []Package {
    i.db.Unscoped().Delete(Package{}, "updated_at < date('now', '-1 day')") // cleanup
    var pckgs []Package
    i.db.Where("Filename LIKE ?", name).Limit(200).Preload("Release").Find(&pckgs)
	/*for indx, _ := range(pckgs) {
		// enrich with parsed release info
		pckgs[indx].Parsed = i.getReleaseForPackage(pckgs[indx])
	}*/
    return pckgs

}

func (i *Indexer) SetupDB() {
  p := path.Join(os.Getenv("HOME"), ".indexer.db")
  db, err := gorm.Open("sqlite3", p)
  if err != nil {
    panic("failed to connect database")
  }
  i.db = db
  db.AutoMigrate(&Package{})
  db.AutoMigrate(&Release{})
  db.AutoMigrate(&Downloaded{})
}

func (indx* Indexer) WaitForPackages(ch chan PrivMsg) {
    listingRegexp := regexp.MustCompile(`(#[0-9]*)[^0-9]*([0-9]*x)[^\[]*\[([ 0-9.]+(?:M|G)?)\][^\x21-\x7E]*(.*)`)
	colorRegexp := regexp.MustCompile(`\x03[0-9,]*([^\x03]*)\x03`)
    for {
        select {
            case msg := <-ch:
                if listingRegexp.MatchString(msg.Content) {


                    matches := listingRegexp.FindStringSubmatch(colorRegexp.ReplaceAllString(msg.Content, "$1"))
                    nmb, gets, size, name := matches[1], matches[2], strings.Trim(matches[3]," \r\n\u000f"), strings.Trim(matches[4], " \r\n\u000f")
                    indx.AddPackage(Package{Server: msg.Server, Channel: msg.To, Bot: msg.From, Package: nmb, Filename: name, Size: size, Gets: gets, Time: time.Now().Format(time.RFC850)})

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
    indx := Indexer{Conf: c, connPool: connPool}
    indx.SetupDB()

	indx.imdb = CreateIMDB(c)

    indx.announcementCh = make(chan PrivMsg, 100)
    for _,el := range c.Channels {
		if !indx.setupChannelListener(el.Server, el.Channel) {
			//return nil
		}
    }

    go indx.WaitForPackages(indx.announcementCh)

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
