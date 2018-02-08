package main

import "testing"
import "fmt"

func BenchmarkAddPackage(b *testing.B) {
    c := Config{}
    c.LoadConfig()
    indx := CreateIndexer(&c)
    b.ResetTimer()
    fmt.Println(b.N)
    for n := 0; n < b.N; n++ {
        indx.AddPackage(Package{Server:"SomeServer", Channel:"SomeChannel", Bot:"SomeBot", Package:"SomePackage"})
    }
    // 125s without index
    // 25s with combined index
}
