package main
import "time"
import "log"
import "strings"
import "regexp"
import "os"
import "path"
import (
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
	 "github.com/cytec/releaseparser"
)


type Indexer struct {
    Conf *Config
    db *gorm.DB
    connPool *ConnectionPool
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
	Parsed releaseparser.Release
}

type Release struct {
	releaseparser.Release
	gorm.Model
}

func (p *Package) Parse() releaseparser.Release {
	return *releaseparser.Parse(p.Filename)
}

func (i *Indexer) getReleaseForPackage(p Package) Release {
	var release Release
	parsed := (p.Parse())
    i.db.Where(&parsed).FirstOrCreate(&release)
	log.Print(release)
	return release	
}

func (i *Indexer) AddPackage(p Package) {
	p.ReleaseID = i.getReleaseForPackage(p).ID
    if !i.UpdateIfExists(p) {
        i.db.Create(&p)
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

func (i *Indexer) Search(name string) []Package {
    i.db.Unscoped().Delete(Package{}, "updated_at < date('now', '-1 day')") // cleanup
    var pckgs []Package
    i.db.Where("Filename LIKE ?", name).Limit(200).Find(&pckgs)
	for indx, _ := range(pckgs) {
		// enrich with parsed release info
		pckgs[indx].Parsed = i.getReleaseForPackage(pckgs[indx])
	}
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
