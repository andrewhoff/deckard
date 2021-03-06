package deckard

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

// BgRunner - this allows the caller to simply define their own implementation and have that be run
type BgRunner struct {
	Concurrency int
	Runnable    BgRunnable
	Done        chan os.Signal
	Total       int
}

// BgRunnable - this allows the caller to simply define their own implementation and have that be run
type BgRunnable interface {
	Run() error
}

const defaultConcurrency = 1

// NewRunner - returns an infinitely runnable BgRunner
func NewRunner(c int, r BgRunnable, done chan os.Signal) BgRunner {
	if c <= 0 {
		c = defaultConcurrency

		maxProcs := os.Getenv("GOMAXPROCS")
		if maxProcs != "" {
			max, err := strconv.Atoi(maxProcs)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("setting concurrency rate to GOMAXPROCS of %d\n", max)
			c = max
		} else {
			fmt.Printf("GOMAXPROCS is unset, using default concurrency of %d\n", defaultConcurrency)
		}
	}

	return BgRunner{
		Concurrency: c,
		Runnable:    r,
		Done:        done,
	}
}

// NewFiniteRunner - returns a BgRunner with its total run count set to `n`
func NewFiniteRunner(c int, r BgRunnable, n int, done chan os.Signal) BgRunner {
	newRunner := NewRunner(c, r, done)
	newRunner.Total = n

	return newRunner
}

// LaunchAndWait - This call blocks either an os.Signal is received on the done channel, or until `n` iterations run to completion
func (r BgRunner) LaunchAndWait() {
	throttle := make(chan int, r.Concurrency)

	if r.Total == 0 { // run until kill signal
		for {
			select {
			case <-r.Done:
				fmt.Println("cleaning up....")
				return // TODO: cleanup goroutines
			default:
				go func() {
					throttle <- 1
					if err := r.Runnable.Run(); err != nil {
						fmt.Printf("error in goroutine: %v", err)
					}
					<-throttle
				}()
			}
		}
	} else {
		wg := sync.WaitGroup{}

		for i := 0; i < r.Total; i++ {
			select {
			case <-r.Done:
				fmt.Println("Done!")
				return // TODO: cleanup goroutines
			default:
				wg.Add(1)

				go func() {
					throttle <- 1
					if err := r.Runnable.Run(); err != nil {
						fmt.Printf("error in goroutine: %v", err)
					}
					<-throttle
					wg.Done()
				}()
			}
		}

		fmt.Println("waiting on background goroutines to finish...")
		wg.Wait()
	}
}
