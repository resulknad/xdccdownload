package main
import "time"
import "fmt"
import "strings"
import "regexp"
import "os"
import "path"
import (
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
)


type Indexer struct {
    Conf *Config
    db *gorm.DB
    connPool *ConnectionPool
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
}

func (i *Indexer) AddPackage(p Package) {
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
    fmt.Println(name)
    var users []Package
    cnt := 0
    i.db.Where("Filename LIKE ?", name).Find(&users)
    for _, el := range users {
        fmt.Println(el)
        cnt++
    }
    return users

}

func (i *Indexer) PrintAll() {
    cnt := 0
    var users []Package
    i.db.Find(&users)
    for _, el := range users {
        fmt.Println(el)
        cnt++
    }
    fmt.Println(cnt)
}

func (i *Indexer) SetupDB() {
  p := path.Join(os.Getenv("HOME"), ".indexer.db")
  db, err := gorm.Open("sqlite3", p)
  if err != nil {
    panic("failed to connect database")
  }
  i.db = db
  db.AutoMigrate(&Package{})
}

func (indx* Indexer) WaitForPackages(ch chan PrivMsg) {
    listingRegexp := regexp.MustCompile(`(#[0-9]*)[^0-9]*([0-9]*x)[^\[]*\[([ 0-9.]+(?:M|G)?)\][^\x21-\x7E]*(.*)`)
	colorRegexp := regexp.MustCompile(`\x03[0-9,]*([^\x03]*)\x03`)
    for {
        select {
            case msg := <-ch:
				fmt.Println(msg)
                if listingRegexp.MatchString(msg.Content) {


                    matches := listingRegexp.FindStringSubmatch(colorRegexp.ReplaceAllString(msg.Content, "$1"))
                    nmb, gets, size, name := matches[1], matches[2], strings.Trim(matches[3]," \r\n\u000f"), strings.Trim(matches[4], " \r\n\u000f")
                    indx.AddPackage(Package{Server: msg.Server, Channel: msg.To, Bot: msg.From, Package: nmb, Filename: name, Size: size, Gets: gets, Time: time.Now().Format(time.RFC850)})

                }
        }
    }
}

func CreateIndexer(c *Config, connPool *ConnectionPool) *Indexer {
        indx := Indexer{Conf: c, connPool: connPool}
    indx.SetupDB()

    ch := make(chan PrivMsg, 100)
    for _,el := range c.Channels {
        i := connPool.GetConnection(el.Server)
        suc := false
        for a:= 0; a<3&&!suc; a++ {
            suc = /*i.Connect() &&*/ i.JoinChannel(el.Channel)
            time.Sleep(time.Duration(0*a)*time.Second)
        }

        if !suc {
            fmt.Println("Coulndt connect to ", el.Server, el.Channel)
        }

        i.SubscriptionCh<-PrivMsgSubscription{Once:false, Backchannel: ch, To:el.Channel}
    }

    go indx.WaitForPackages(ch)

    return &indx
}
