package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/rawbytedev/fractus"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	runtime.MemProfileRate = 1
	type NewStruct struct {
		Val      []string
		Mod      []int8
		Integers []int16
		Float3   []float32
		Float6   []float64
	}
	Val := []string{"azerty", "hello", "world", "random"}
	z := NewStruct{Val: Val,
		Mod: []int8{12, 10, 13, 0}, Integers: []int16{100, 250, 300},
		Float3: []float32{12.13, 16.23, 75.1}, Float6: []float64{100.5, 165.63, 153.5}}
	y := fractus.NewFractus(fractus.SafeOptions{UnsafeStrings: true, UnsafePrimitives: true})
	for i := 0; i < 10000; i++ {
		data, _ := y.Encode(z)
		res := &NewStruct{}
		y.Decode(data, res)
	}
	pprof.WriteHeapProfile(f)
	time.Sleep(5 * time.Minute)
}
