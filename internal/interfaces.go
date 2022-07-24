package internal

type ServerProxy interface {
	Lock(name string)
	Unlock(name string)
}
