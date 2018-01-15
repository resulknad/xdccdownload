package main
import "time"
import "fmt"
import "os"
import "bufio"
import "strings"
import "regexp"
import "github.com/HouzuoGuo/tiedot/db"
//import "github.com/HouzuoGuo/tiedot/dberr"
import "github.com/fatih/structs"
import "encoding/json"



type Indexer struct {
    Packages *db.Col
}

type Package struct {
    Server string
    Channel string
    Bot string
    Package string
    Filename string
    Size string
    Gets string
    Time string
}

func (i *Indexer) AddPackage(p Package) {
    if i.RemoveIfExists(p) {
    i.Packages.Insert(structs.Map(p))
}
}

func (i *Indexer) RemoveIfExists(p Package) bool {
    var query interface{}
    json.Unmarshal([]byte(fmt.Sprintf(`{"n":[{"eq": "%s", "in": ["Server"]}, {"eq": "%s", "in": ["Bot"]}, {"eq": "%s", "in": ["Package"]}, {"eq": "%s", "in": ["Channel"]}]}`, p.Server, p.Bot, p.Package, p.Channel)), &query)

	queryResult := make(map[int]struct{}) // query result (document IDs) goes into map keys
	if err := db.EvalQuery(query, i.Packages, &queryResult); err != nil {
        panic(err)
    }
    for id := range queryResult {
        i.Packages.Update(id, structs.Map(p))
//        i.Packages.Delete(id)
        fmt.Println("Deleted, bc was already in database ",id)
        return false
    }
    return true
}

func (i *Indexer) Search(name string) {

	i.Packages.ForEachDoc(func(id int, docContent []byte) (willMoveOn bool) {
        var pkg Package
        json.Unmarshal(docContent, &pkg)
        if (strings.Contains(strings.ToLower(pkg.Filename), name)) {
            fmt.Println("Document", id, "is", string(docContent))
        }
        return true  // move on to the next document OR
    })
}

func (i *Indexer) PrintAll() {
    // Process all documents (note that document order is undetermined)
    cnt := 0
	i.Packages.ForEachDoc(func(id int, docContent []byte) (willMoveOn bool) {
        fmt.Println("Document", id, "is", string(docContent))
        cnt++
        return true  // move on to the next document OR
    })
    fmt.Println(cnt)
}

func (i *Indexer) SetupDB() {
    dbDir := "./database/"
    myDB, err := db.OpenDB(dbDir)
    if err != nil {
        panic(err)
    }

    if err := myDB.Create("Packages"); err != nil {
        fmt.Println(err)
    }
    if err := myDB.Scrub("Packages"); err != nil {
		panic(err)
    }

    i.Packages = myDB.Use("Packages")
	if err := i.Packages.Index([]string{"Server"}); err != nil {
		fmt.Println(err)
    }
	if err := i.Packages.Index([]string{"Bot"}); err != nil {
		fmt.Println(err)
    }
	if err := i.Packages.Index([]string{"Package"}); err != nil {
		fmt.Println(err)
    }
	if err := i.Packages.Index([]string{"Channel"}); err != nil {
		fmt.Println(err)
    }
}

func (indx* Indexer) WaitForPackages(ch chan PrivMsg) {
    listingRegexp := regexp.MustCompile(`(#[0-9]*)[^0-9]*([0-9]*x)[^\[]*\[([ 0-9.]+(?:M|G)?)\][^\x21-\x7E]*(.*)`)
    for {
        select {
        case msg := <-ch:
            if listingRegexp.MatchString(msg.Content) {
                matches := listingRegexp.FindStringSubmatch(msg.Content)
                nmb, gets, size, name := matches[1], matches[2], strings.Trim(matches[3]," \r\n\u000f"), strings.Trim(matches[4], " \r\n\u000f")
                indx.AddPackage(Package{Server: msg.Server, Channel: msg.Channel, Bot: msg.From, Package: nmb, Filename: name, Size: size, Gets: gets, Time: time.Now().Format(time.RFC850)})

            }
        default: // wait for multiple messages before inserting
        fmt.Println("sleep")
            time.Sleep(10*time.Second)
    }
}
}

func CreateIndexer() *Indexer {
    f, _ := os.Open("channels.txt")
    scanner := bufio.NewScanner(f)

    indx := Indexer{}
    indx.SetupDB()
    indx.PrintAll()
    indx.Search("j._cole")
    time.Sleep(10 * time.Second)

    ch := make(chan PrivMsg, 100)
    for scanner.Scan() {
        line := strings.Split(scanner.Text(), " ")
        if len(line) != 2 {
            fmt.Println("coulnd parse line...")
            continue;
        }

        i := IRC{Server: line[1]}
        suc := false
        for a:= 0; a<5&&!suc; a++ {
            suc = i.Connect() && i.JoinChannel(line[0]) 
            time.Sleep(time.Duration(0*a)*time.Second)
        }
        if !suc {
            fmt.Println("Coulndt connect to ", line[0], line[1])
        }


        i.SubscriptionCh<-PrivMsgSubscription{Once:false, Backchannel: ch, To:line[0]}

    }
    go indx.WaitForPackages(ch)

    return &indx
}
