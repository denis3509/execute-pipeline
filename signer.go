package main

import (
	"execute-pipeline/logs"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var maxWorkers = 100
var chanBuffer = 100
var md5Quota = 1

func ExecutePipeline(jobs ...job) {
	prevOut := make(chan interface{}, chanBuffer)
	wg := &sync.WaitGroup{}
	wg.Add(len(jobs))
	for _, jobFn := range jobs {
		out := make(chan interface{}, chanBuffer)
		go func(jobFn job, in, out chan interface{}) {
			jobFn(in, out)
			close(out)
			wg.Done()
		}(jobFn, prevOut, out)
		prevOut = out
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	md5Holder := make(chan struct{}, md5Quota)
	worker := func() {
		for dataRaw := range in {
			data := fmt.Sprint(dataRaw)
			logs.Debug("'received'", data)
			md5 := make(chan string, 1)
			left := make(chan string, 1)
			right := make(chan string, 1)
			go func() {
				md5Holder <- struct{}{}
				md5 <- DataSignerMd5(data)
				<-md5Holder
			}()
			go func() {
				right <- DataSignerCrc32(<-md5)
			}()
			go func() {
				left <- DataSignerCrc32(data)
			}()
			concat := strings.Join([]string{<-left, <-right}, "~")
			logs.Warning("'single hash res'", concat)
			out <- concat
		}
		wg.Done()
	}
	wg.Add(maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		go worker()
	}
	wg.Wait()
}
func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	worker := func() {
		for dataRaw := range in {
			data := fmt.Sprint(dataRaw)
			wg := &sync.WaitGroup{}
			wg.Add(6)
			hashMap := map[int]string{}
			hashMapLocker := &sync.Mutex{}
			for i := 0; i < 6; i++ {
				go func(num int) {
					concat := strings.Join([]string{strconv.Itoa(num), data}, "")
					res := DataSignerCrc32(concat)
					runtime.Gosched()
					hashMapLocker.Lock()
					hashMap[num] = res
					hashMapLocker.Unlock()
					wg.Done()
				}(i)
			}
			wg.Wait()
			concat := ""
			for i := 0; i < 6; i++ {
				concat = strings.Join([]string{concat, hashMap[i]}, "")
				runtime.Gosched()
			}
			out <- concat
		}
		wg.Done()
	}
	wg.Add(maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		go worker()
	}
	wg.Wait()
}
func CombineResults(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	hashes := make([]string, 0)
	hashesLocker := &sync.Mutex{}
	worker := func() {
		for dataRaw := range in {
			data := fmt.Sprint(dataRaw)
			hashesLocker.Lock()
			hashes = append(hashes, data)
			hashesLocker.Unlock()
		}
		wg.Done()
	}
	wg.Add(maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		go worker()
	}
	wg.Wait()
	sort.Strings(hashes)
	out <- strings.Join(hashes, "_")
}
