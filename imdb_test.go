package main

import "testing"
import "os"
import "path"
import "fmt"

func CIMDB() *IMDB {
	c := (Config{})
	c.TempPath = "tmp/"
	os.Mkdir("tmp", os.ModePerm)
	i := IMDB{conf:&c}
	i.SetupDB()
	return &i
}

func TestDownload(t *testing.T) {
	i := CIMDB()	

	i.downloadData()
	
	expectedFiles := []string{"title.basics.tsv.gz", "title.ratings.tsv.gz"}
	for _,f := range expectedFiles {
		if _, err := os.Stat(path.Join(i.conf.TempPath, f)); err != nil {
			t.FailNow()
		  }
	}
}

func TestUpdateIMDB(t *testing.T) {
	return
	i := CIMDB()
	i.UpdateData()
}

func TestIMDBAccuracy(t *testing.T) {

	i := CIMDB()
	idd := i.GetIdForMovie("", 0)
	fmt.Println(idd)
	if idd != "" {
		t.FailNow()
	}
	rating, num := i.GetRating(i.GetIdForMovie("Interstellar", 2014))
	if rating < 8 || num <100 {
		t.FailNow()
	}
}

