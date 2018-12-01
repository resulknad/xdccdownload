package main 

import "net/http"
import "compress/gzip"
import "io"
import "os"
import "path"
import "fmt"
import "bufio"
import "strconv"
import "strings"
import "log"

import (
     "github.com/boltdb/bolt"
)

func CreateIMDB(c *Config) *IMDB{
	i := IMDB{conf: c}
	i.SetupDB()
	return &i
}

type IMDB struct {
	conf *Config
	db *bolt.DB
	tx *bolt.Tx
}
/*

    Server string `gorm:"index:pckg"`
    Channel string `gorm:"index:pckg"`
    Bot string `gorm:"index:pckg"`
    Package string `gorm:"index:pckg"`
	*/
type TitleBasic struct {
	Tconst string `gorm:"index"`
	TitleType string `gorm:"index:tt,mv,show"`
	PrimaryTitle string `gorm:"index:pt,mv,show"`
	OriginalTitle string
	IsAdult string
	StartYear int `gorm:"index:sy,mv"`
	EndYear int
	RuntimeMinutes int
	Genres string
}


type TitleRating struct {
	Tconst string `gorm:"index"`
	AverageRating float64
	NumVotes int
}

func (i *IMDB) GetIdForMovie(title string, year int) string {
  v := ""
  i.db.View(func(tx *bolt.Tx) error {
	b := tx.Bucket([]byte("imdbid"))
	v = fmt.Sprintf("%s",b.Get([]byte(i.key("movie", year, title))))
	return nil
  })
  return v
}

func (i *IMDB) GetIdForShow(title string) string {
  v := ""
  i.db.View(func(tx *bolt.Tx) error {
	b := tx.Bucket([]byte("imdbid"))
	v = fmt.Sprintf("%s", b.Get([]byte(i.key("tvSeries", 0, title))))
	return nil
  })
  return v
}

func (i *IMDB) GetRating(id string) (float64, int) {
  v := 0.0
  i.db.View(func(tx *bolt.Tx) error {
	b := tx.Bucket([]byte("rating"))
	v,_ = strconv.ParseFloat(fmt.Sprintf("%s", b.Get([]byte(id))), 64)
	return nil
  })
  return v, 0
}

func (i *IMDB) key(kind string, year int, title string) string {
  if kind == "movie" {
	return fmt.Sprintf("%s:%d:%s", kind, year, strings.ToLower(title))
  } else {
	return fmt.Sprintf("%s:%s", kind, strings.ToLower(title))
  }
}

func (i *IMDB) SetupDB() {
  fmt.Println("setting up db")
  p := path.Join(i.conf.DBPath, ".imdb.bdb")
  db, err := bolt.Open(p, 0600, nil)
  if err != nil {
    panic("failed to connect database")
  }

  db.Update(func(tx *bolt.Tx) error {
	  tx.CreateBucketIfNotExists([]byte("imdbid"))
	  tx.CreateBucketIfNotExists([]byte("rating"))
	return nil
  })
  i.db = db
  fmt.Println("done setting up db")
}

func (i *IMDB) UpdateData() (suc bool) {
	log.Print("Updating data")
	suc = true
	i.downloadData()
	err := i.db.Update(func(tx *bolt.Tx) error {
	  tx.DeleteBucket([]byte("imdbid"))
	  tx.CreateBucketIfNotExists([]byte("imdbid"))
	  tx.DeleteBucket([]byte("rating"))
	  tx.CreateBucketIfNotExists([]byte("rating"))
	  i.tx = tx
	  i.readTSV(path.Join(i.conf.TempPath, "title.basics.tsv.gz"), handleTitleBasics)
	  i.readTSV(path.Join(i.conf.TempPath, "title.ratings.tsv.gz"), handleTitleRatings)
	  return nil
	})

	if err != nil {
	  panic(err)
	}
	log.Print("Done updating data")
	return suc
}


func (i* IMDB) downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil  {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		return err
	}

	return nil
}

func (i *IMDB) downloadData() {
  basePath := "http://localhost:8000/"
	files := []string{"title.basics.tsv.gz", "title.ratings.tsv.gz"}
	for _,f := range files {
		targetFile := path.Join(i.conf.TempPath,f)
		err := i.downloadFile(targetFile, basePath + f)
		if err != nil {
			panic(err)
		}
	}
}

func handleTitleBasics(imdbid *bolt.Bucket, ratings *bolt.Bucket,i *IMDB,vals []string) {
	StartYear, _:= strconv.Atoi(vals[5])	
	  imId := vals[0]
	  TitleType := vals[1]
	  if TitleType == "movie" || TitleType == "tvSeries" {	
		err := imdbid.Put([]byte(i.key(TitleType,StartYear, cleanTitle(vals[2]))), []byte(imId))
		if err != nil {
		  panic(err)
		}
	  }
}

func handleTitleRatings(imdbid *bolt.Bucket, ratings *bolt.Bucket, i *IMDB,vals []string) {
	err := ratings.Put([]byte(vals[0]), []byte(vals[1]))
	if err != nil {
	  panic(err)
	}
}

func (i *IMDB) readTSV(file string, handle func(*bolt.Bucket, *bolt.Bucket,*IMDB, []string)) {
	f, _ := os.Open(file)
	reader, _ := gzip.NewReader(f)
	scanner := bufio.NewScanner(reader)
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		imdbid := i.tx.Bucket([]byte("imdbid"))
		ratings := i.tx.Bucket([]byte("rating"))
		handle(imdbid, ratings, i, strings.Split(strings.Replace(scanner.Text(), "\\N", "", -1), "\t"))
	}
}


// nned to export from releaseparser at some point
func cleanTitle(name string) string {
	name = strings.Replace(name, ".", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Trim(name, " ")
	return name
}
