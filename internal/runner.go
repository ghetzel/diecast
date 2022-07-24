package internal

type Runner interface {
	Init() error
	SetFunction(name string, fn interface{})
	HandleMessage(req *SandboxMessage) *SandboxMessage
}
