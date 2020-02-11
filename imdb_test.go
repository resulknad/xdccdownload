package main

import "testing"
import "os"
import "fmt"

func CIMDB() *IMDB {
	c := (Config{})
	c.TempPath = "tmp/"
	c.DBPath = "tmp/"
	os.Mkdir("tmp", os.ModePerm)
	i := IMDB{conf:&c}
	i.SetupDB()
	return &i
}

func TestDownload(t *testing.T) {
  /*
  fmt.Println("test download")
	i := CIMDB()	

	i.downloadData()
	
	expectedFiles := []string{"title.basics.tsv.gz", "title.ratings.tsv.gz"}
	for _,f := range expectedFiles {
		if _, err := os.Stat(path.Join(i.conf.TempPath, f)); err != nil {
			t.FailNow()
		  }
	}
	*/
}

func TestUpdateIMDB(t *testing.T) {
  return
  fmt.Println("test update")
	i := CIMDB()
	i.UpdateData()
	i.db.Close()
}

func TestIMDBAccuracy(t *testing.T) {
  fmt.Println("test accuracy")

	i := CIMDB()
	/*idd := i.GetIdForMovie("", 0)
	fmt.Println(idd)
	if idd != "" {
		t.FailNow()
	}*/
	rating, _ := i.GetRating(i.GetIdForShow("Narcos"))
	  fmt.Println(rating)
	if rating < 8  {

	  panic("")
		t.FailNow()
	}
}

