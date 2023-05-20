
package pkg

import (
	_ "fmt"
	"strconv"
	"time"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {

	s := make(semaphore, 1)

	s.P()
	s.V()

}

func add(t *testing.T, list *[]int, lock semaphore, doneCh chan bool, i int) {
	lock.P()

	if i == 0 {
		pause, err := time.ParseDuration("100ms")
		if err != nil {
			t.Fatalf("Failed to parse duration: %s", err)
		}
		time.Sleep(pause)
	}

	*list = append(*list, i)
	
	lock.V()
	doneCh <- true
}

func TestOrder(t *testing.T) {

	lock := make(semaphore, 1)
	var list *[]int

	list = &[]int{}

	launchInterval, err := time.ParseDuration("10ms")
	if err != nil {
		t.Fatalf("Error when parsing duration: %s\n", err)
	}

	doneChs := [](chan bool){}
	for i := 0; i < 10; i++ {
		doneCh := make(chan bool)
		doneChs = append(doneChs, doneCh)
		go add(t, list, lock, doneCh, i)
		if i < 9 {
			time.Sleep(launchInterval)
		}
		t.Logf("Launching %d\n", i)
	}

	// wait for all goroutines
	for _, ch := range doneChs {
		<-ch
	}

	result := ""
	for _, i := range *list {
		result = result + strconv.Itoa(i)
	}

	assert.Equal(t, "0123456789", result)
}

//
// Without working write lock, readers don't find "mike" in the list
//
func TestReadWriteWriterCanLock(t *testing.T) {

	lock := &ReadWriteLock{}
	lock.Init()
	
	writeAcquired := make(chan bool)
	list := []string{"hello"}

	g1 := make(chan bool)
	
	go func(lock *ReadWriteLock, list *[]string) {
		lock.LockForWriting()
		writeAcquired <- true

		// wait for readers to get in line
		pause, _ := time.ParseDuration("10ms")
		time.Sleep(pause)
		
		*list = append(*list, "mike")

		// readers can then proceed
		lock.WritingUnlock()
		g1 <- true
	}(lock, &list)

	<-writeAcquired

	readers := [](chan bool){}
	for i := 0; i < 3; i++ {
		nextCh := make(chan bool)
		readers = append(readers, nextCh)
		go func(lock *ReadWriteLock, list *[]string) {
			lock.LockForReading()

			assert.Equal(t, []string{"hello", "mike"}, *list)
			
			lock.ReadingUnlock()
			nextCh <- true
		}(lock, &list)
	}

	for _, ch := range readers {
		<-ch
	}
	<-g1
}


//
// If readers can't block writer, they find "mike" in the list
//
func TestReadWriteReadersCanLock(t *testing.T) {
	lock := &ReadWriteLock{}
	lock.Init()
	
	list := []string{"hello"}

	readLockAcquired := [](chan bool){
		make(chan bool),
		make(chan bool),
		make(chan bool),
	}

	rDone := [](chan bool){
		make(chan bool),
		make(chan bool),
		make(chan bool),
	}

	g2 := make(chan bool)
	
	acquireReadLock := func(lock *ReadWriteLock, list *[]string, idx int) {
		lock.LockForReading()
		readLockAcquired[idx] <- true

		// let writer get stuck
		pause, _ := time.ParseDuration("10ms")
		time.Sleep(pause)
		
		assert.Equal(t, []string{"hello"}, *list)
		
		lock.ReadingUnlock()
		rDone[idx] <- true
	}
	
	for i := 0; i < 3; i+= 1 {
		go acquireReadLock(lock, &list, i)
	}

	for i := 0; i < 3; i+= 1 {
		<-readLockAcquired[i]
	}
	
	go func(lock *ReadWriteLock, list *[]string) {
		lock.LockForWriting()
		
		*list = append(*list, "mike")

		// readers can then proceed
		lock.WritingUnlock()
		g2 <- true
	}(lock, &list)

	for i := 0; i < 3; i+= 1 {
		<-rDone[i]
	}
	<-g2
}
