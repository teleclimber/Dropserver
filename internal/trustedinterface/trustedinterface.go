package trustedinterface

// TrustedInterface provides an RPC interface to ds-trusted
type TrustedInterface interface {
	CreateSandboxDir(sandboxName string)
	DeleteSandboxDir(sandboxName string)
}
