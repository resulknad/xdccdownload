package main

import "testing"
import "fmt"
func Indx() *Indexer {
	connPool := ConnectionPool{}
	chs := []ChannelConfig{ChannelConfig{Server:"irc.abjects.net:6667", Channel:"#aaaaasdf"}}
	c := (Config{})
	c.Channels = chs

	indx := CreateIndexer(&c, &connPool)
	return indx
}

func BenchmarkAddPackage(b *testing.B) {
    /*c := Config{}
    c.LoadConfig()
    indx := CreateIndexer(&c)
    b.ResetTimer()
    fmt.Println(b.N)
    for n := 0; n < b.N; n++ {
        indx.AddPackage(Package{Server:"SomeServer", Channel:"SomeChannel", Bot:"SomeBot", Package:"SomePackage"})
    }*/
    // 125s without index
    // 25s with combined index
}

func TestAddRelease(t *testing.T) {
	indx := Indx()	
	indx.AddPackage(Package{Filename: "Von.Maeusen.und.Menschen.GERMAN.1992.AC3.BDRip.x264-UNiVERSUM"})
	// todo test smth
}

func TestSizeConv(t *testing.T) {
	if (&Package{Size: "1.9G "}).SizeMbytes() != 1.9*1024 {
		t.FailNow()
	}
	if (&Package{Size: "2G "}).SizeMbytes() != 2*1024 {
		t.FailNow()
	}
	if (&Package{Size: "2M  "}).SizeMbytes() != 2 {
		t.FailNow()
	}
	if (&Package{Size: "2  "}).SizeMbytes() != -1 {
		t.FailNow()
	}
}
func TestAddDownloaded(t *testing.T) {
	indx := Indx()	
	indx.AddDownloaded(Package{Filename: "Von.Maeusen.und.Menschen.GERMAN.1992.AC3.BDRip.x264-UNiVERSUM"})
	if !indx.CheckDownloaded(Package{Filename: "Von.Maeusen.und.Menschen.GERMAN.1992.AC3.BDRip.x264-UNiVERSUM"}) {
		fmt.Print("1")
		t.FailNow()
	}
	if indx.CheckDownloaded(Package{Filename: "Movie.Which.Doesnt.Exist.2018-test"}) {
		fmt.Print("2")
		t.FailNow()
	}
	if !indx.CheckDownloadedExact(Package{Filename: "Von.Maeusen.und.Menschen.GERMAN.1992.AC3.BDRip.x264-UNiVERSUM"}) {
		t.FailNow()
	}
	if indx.CheckDownloadedExact(Package{Filename: "Von.Maeusen.und.Menschen.GERMAN.1992.AC3.BDRip.x265-UNiVERSUM"}) {
		t.FailNow()
	}

}
