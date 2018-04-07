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
     "github.com/jinzhu/gorm"
     _"github.com/jinzhu/gorm/dialects/sqlite"
)

func CreateIMDB(c *Config) *IMDB{
	i := IMDB{conf: c}
	i.SetupDB()
	return &i
}

type IMDB struct {
	conf *Config
	db *gorm.DB
	tx *gorm.DB
}
/*

    Server string `gorm:"index:pckg"`
    Channel string `gorm:"index:pckg"`
    Bot string `gorm:"index:pckg"`
    Package string `gorm:"index:pckg"`
	*/
type TitleBasic struct {
	gorm.Model
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
	gorm.Model
	Tconst string `gorm:"index"`
	AverageRating float64
	NumVotes int
}

func (i *IMDB) GetIdForMovie(title string, year int) string {
	var movie TitleBasic
	i.db.Where("title_type = ? AND primary_title = ? and start_year = ?", "movie", title, year).First(&movie)
	//i.db.Where(TitleBasic{TitleType: "movie", PrimaryTitle: title, StartYear: year}).First(&movie)
	if title != "" && movie.ID != 0 {
		return movie.Tconst
	} else {
		return ""
	}
}

func (i *IMDB) GetIdForShow(title string) string {
	var movie TitleBasic
	i.db.Where("title_type = ? AND primary_title = ?", "tvSeries", title).First(&movie)
	//i.db.Where(TitleBasic{TitleType: "tvSeries", PrimaryTitle: title}).First(&movie)
	if title != "" && movie.ID != 0 {
		return movie.Tconst
	} else {
		return ""
	}
}

func (i *IMDB) GetRating(id string) (float64, int) {
	var rating TitleRating
	i.db.Where(TitleRating{Tconst: id}).First(&rating)
	return rating.AverageRating, rating.NumVotes
}

func (i *IMDB) SetupDB() {
  p := path.Join(os.Getenv("HOME"), ".imdb.db")
  db, err := gorm.Open("sqlite3", p)
  if err != nil {
    panic("failed to connect database")
  }
  i.db = db
  db.AutoMigrate(&TitleBasic{})
  db.AutoMigrate(&TitleRating{})
}

func (i *IMDB) UpdateData() (suc bool) {
	log.Print("Updating data")
	defer func() {
		if recover() != nil {
			suc = false
		}
	}()
	suc = true
	i.downloadData()
	i.tx = i.db.Begin() // create transaction
	i.tx.Exec("DELETE FROM title_ratings; DELETE FROM title_basics;") // delete everything

	i.readTSV(path.Join(i.conf.TempPath, "title.basics.tsv.gz"), handleTitleBasics)
	i.readTSV(path.Join(i.conf.TempPath, "title.ratings.tsv.gz"), handleTitleRatings)
	if i.tx.Commit().Error != nil {
		suc = false
	}
	i.db.Exec("VACUUM;") // make sqlite actually delete stuff
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
	basePath := "https://datasets.imdbws.com/"
	files := []string{"title.basics.tsv.gz", "title.ratings.tsv.gz"}
	for _,f := range files {
		targetFile := path.Join(i.conf.TempPath,f)
		err := i.downloadFile(targetFile, basePath + f)
		if err != nil {
			panic(err)
		}
	}
}

func handleTitleBasics(i *IMDB,vals []string) {
	StartYear, err1:= strconv.Atoi(vals[5])
	EndYear, err2 := strconv.Atoi(vals[6])
	RuntimeMinutes, err3 := strconv.Atoi(vals[7])
	
	if (err1 == nil || vals[5] == "") &&
	   (err2 == nil || vals[6] == "") &&
	   (err3 == nil || vals[7] == "") {
		t := TitleBasic{}
		t.Tconst = vals[0]
		t.TitleType = vals[1]
		t.PrimaryTitle = cleanTitle(vals[2])
		t.OriginalTitle = cleanTitle(vals[3])
		t.IsAdult = vals[4]
		t.StartYear = StartYear
		t.EndYear = EndYear
		t.RuntimeMinutes = RuntimeMinutes
		t.Genres = vals[8]
		if t.TitleType == "movie" || t.TitleType == "tvSeries" {
			i.tx.Create(&t)	
		}
	} else {
		fmt.Print(vals)
		panic("int parse error")
	}
}

func handleTitleRatings(i *IMDB,vals []string) {
	AvgRating, err1:= strconv.ParseFloat(vals[1], 64)
	NumVotes, err2 := strconv.Atoi(vals[2])
	
	if (err1 == nil) &&
	   (err2 == nil) {
		t := TitleRating{}
		t.Tconst = vals[0]
		t.AverageRating = AvgRating
		t.NumVotes = NumVotes
		i.tx.Create(&t)	
	} else {
		fmt.Print(vals)
		panic("int parse error")
	}
}

func (i *IMDB) readTSV(file string, handle func(*IMDB, []string)) {
	f, _ := os.Open(file)
	reader, _ := gzip.NewReader(f)
	scanner := bufio.NewScanner(reader)
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		handle(i, strings.Split(strings.Replace(scanner.Text(), "\\N", "", -1), "\t"))
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
