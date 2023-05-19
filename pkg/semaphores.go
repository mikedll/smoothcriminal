
package pkg

type empty struct {}
type semaphore chan empty

func (s semaphore) P() {
	e := empty{}
	s <- e
}

func (s semaphore) V() {
	<-s
}
