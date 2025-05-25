package internal

type Runner interface {
	Init() error
	SetFunction(name string, fn any)
	HandleMessage(req *SandboxMessage) *SandboxMessage
}
