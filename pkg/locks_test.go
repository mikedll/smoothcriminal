
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

func add(t *testing.T, list *[]int, lock semaphore, doneCh chan Empty, i int) {
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
	doneCh <- Em
}

func TestOrder(t *testing.T) {

	lock := make(semaphore, 1)
	var list *[]int

	list = &[]int{}

	launchInterval, err := time.ParseDuration("10ms")
	if err != nil {
		t.Fatalf("Error when parsing duration: %s\n", err)
	}

	doneChs := [](chan Empty){}
	for i := 0; i < 10; i++ {
		doneCh := make(chan Empty)
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
	
	writeAcquired := make(chan Empty)
	list := []string{"hello"}

	g1 := make(chan Empty)
	
	go func(lock *ReadWriteLock, list *[]string) {
		lock.LockForWriting()
		writeAcquired <- Em

		// wait for readers to get in line
		pause, _ := time.ParseDuration("10ms")
		time.Sleep(pause)
		
		*list = append(*list, "mike")

		// readers can then proceed
		lock.WritingUnlock()
		g1 <- Em
	}(lock, &list)

	<-writeAcquired

	readers := [](chan Empty){}
	for i := 0; i < 3; i++ {
		nextCh := make(chan Empty)
		readers = append(readers, nextCh)
		go func(lock *ReadWriteLock, list *[]string) {
			lock.LockForReading()

			assert.Equal(t, []string{"hello", "mike"}, *list)
			
			lock.ReadingUnlock()
			nextCh <- Em
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

	readLockAcquired := [](chan Empty){
		make(chan Empty),
		make(chan Empty),
		make(chan Empty),
	}

	rDone := [](chan Empty){
		make(chan Empty),
		make(chan Empty),
		make(chan Empty),
	}

	g2 := make(chan Empty)
	
	acquireReadLock := func(lock *ReadWriteLock, list *[]string, idx int) {
		lock.LockForReading()
		readLockAcquired[idx] <- Em

		// let writer get stuck
		pause, _ := time.ParseDuration("10ms")
		time.Sleep(pause)
		
		assert.Equal(t, []string{"hello"}, *list)
		
		lock.ReadingUnlock()
		rDone[idx] <- Em
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
		g2 <- Em
	}(lock, &list)

	for i := 0; i < 3; i+= 1 {
		<-rDone[i]
	}
	<-g2
}

func TestReadWriteWriterBlocksWriter(t *testing.T) {
	lock := &ReadWriteLock{}
	lock.Init()

	firstWriterIsIn := make(chan Empty)
	list := []string{"hello"}

	g1 := make(chan Empty)
	g2 := make(chan Empty)
	
	go func(lock *ReadWriteLock, list *[]string) {
		lock.LockForWriting()
		firstWriterIsIn <- Em

		// wait for other writer to get in line
		pause, _ := time.ParseDuration("10ms")
		time.Sleep(pause)
		
		*list = append(*list, "mike")

		// readers can then proceed
		lock.WritingUnlock()
		g1 <- Em
	}(lock, &list)

	<-firstWriterIsIn

	go func(lock *ReadWriteLock, list *[]string) {
		lock.LockForWriting()
		
		*list = append(*list, "how are you")

		// readers can then proceed
		lock.WritingUnlock()
		g2 <- Em
	}(lock, &list)	

	<-g1
	<-g2

	assert.Equal(t, []string{"hello", "mike", "how are you"}, list)
}
