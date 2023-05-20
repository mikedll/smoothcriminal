
package pkg

type empty struct {}
type semaphore chan empty
type ReadWriteLock struct {
	ServiceQueue semaphore
	ReaderCountLock semaphore
	ReaderCount int
	ResourceLock semaphore
}

func (s semaphore) P() {
	e := empty{}
	s <- e
}

func (s semaphore) V() {
	<-s
}

func (rw *ReadWriteLock) Init() {
	rw.ServiceQueue = make(semaphore, 1)
	rw.ReaderCountLock = make(semaphore, 1)
	rw.ResourceLock = make(semaphore, 1)
}

func (rw *ReadWriteLock) LockForWriting() {
	rw.ServiceQueue.P()
	rw.ResourceLock.P()
	rw.ServiceQueue.V()
}

func (rw *ReadWriteLock) WritingUnlock() {
	rw.ResourceLock.V()
}

func (rw *ReadWriteLock) LockForReading() {
	rw.ServiceQueue.P()
	rw.ReaderCountLock.P()
	rw.ReaderCount += 1
	if rw.ReaderCount == 1 {
		rw.ResourceLock.P()
	}
	rw.ServiceQueue.V()
	rw.ReaderCountLock.V()
}

func (rw *ReadWriteLock) ReadingUnlock() {
	rw.ReaderCountLock.P()
	rw.ReaderCount -= 1
	if rw.ReaderCount == 0 {
		rw.ResourceLock.V()
	}
	rw.ReaderCountLock.V()	
}
