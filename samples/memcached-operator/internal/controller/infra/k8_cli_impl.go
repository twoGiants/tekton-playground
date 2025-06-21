package infra

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8CliImpl is a Thin Wrapper (James Shore) encapsulating the Infrastructure Wrapper
// and Embedded Stub for the k8 client. Its single job is to forward requests.
type K8CliImpl struct {
	cli k8Cli
}

// Package scoped interface which is used by the Thin Wrapper and implemented by
// Infrastructure Wrapper and the Embedded Stub.
type k8Cli interface {
	Get(context.Context, types.NamespacedName, client.Object) error
	StatusUpdate(context.Context, client.Object) error
	Create(context.Context, client.Object) error
	Update(context.Context, client.Object) error
}

func NewK8CliImpl(k8 client.Client) *K8CliImpl {
	return &K8CliImpl{&k8CliActual{k8}}
}

func NewK8CliStub(e StubErrors, k8 client.Client) *K8CliImpl {
	var cli k8Cli
	if k8 != nil {
		cli = &k8CliActual{k8}
	}
	return &K8CliImpl{&k8CliStub{e, cli}}
}

func (k8 *K8CliImpl) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	return k8.cli.Get(ctx, t, co)
}

func (k8 *K8CliImpl) StatusUpdate(ctx context.Context, co client.Object) error {
	return k8.cli.StatusUpdate(ctx, co)
}

func (k8 *K8CliImpl) Create(ctx context.Context, co client.Object) error {
	return k8.cli.Create(ctx, co)
}

func (k8 *K8CliImpl) Update(ctx context.Context, co client.Object) error {
	return k8.cli.Update(ctx, co)
}

// Infrastructure Wrapper which is the real implementation using the k8 client
type k8CliActual struct {
	cli client.Client
}

func (k8 *k8CliActual) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	return k8.cli.Get(ctx, t, co)
}

func (k8 *k8CliActual) StatusUpdate(ctx context.Context, co client.Object) error {
	return k8.cli.Status().Update(ctx, co)
}

func (k8 *k8CliActual) Create(ctx context.Context, co client.Object) error {
	return k8.cli.Create(ctx, co)
}

func (k8 *k8CliActual) Update(ctx context.Context, co client.Object) error {
	return k8.cli.Update(ctx, co)
}

// Configurable Responses. Key: method name, value: error slice.
type StubErrors = map[string][]error

// Embedded Stub with Configurable Responses. It takes a map of error slices and
// returns errors if the key is the name of an existing method. Each returned error
// is removed from the slice. After the last error return it returns nil. If a k8
// client is provided it forwards requests to it, if not it returns nil. Error
// returns have higher precedence over the k8 client.
type k8CliStub struct {
	errArr StubErrors
	cli    k8Cli
}

func (k8 *k8CliStub) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	return k8.do("Get", func() error {
		return k8.cli.Get(ctx, t, co)
	})
}

func (k8 *k8CliStub) do(name string, action func() error) error {
	if nextErr := k8.nextErr(name); nextErr != nil {
		return nextErr
	}

	if k8.cli == nil {
		return nil
	}

	return action()
}

func (k8 *k8CliStub) nextErr(name string) error {
	if k8.errArr == nil {
		fmt.Println("no errors configured in nullable")
		return nil
	}

	if _, ok := k8.errArr[name]; !ok {
		fmt.Printf("no errors configured for '%s' method\n", name)
		return nil
	}

	if len(k8.errArr[name]) == 0 {
		fmt.Printf("no more errors configured in nulled '%s' method\n", name)
		return nil
	}

	err := k8.errArr[name][0]
	k8.errArr[name] = k8.errArr[name][1:]

	return err
}

func (k8 *k8CliStub) StatusUpdate(ctx context.Context, co client.Object) error {
	return k8.do("StatusUpdate", func() error {
		return k8.cli.StatusUpdate(ctx, co)
	})
}

func (k8 *k8CliStub) Create(ctx context.Context, co client.Object) error {
	return k8.do("Create", func() error {
		return k8.cli.Create(ctx, co)
	})
}

func (k8 *k8CliStub) Update(ctx context.Context, co client.Object) error {
	return k8.do("Update", func() error {
		return k8.cli.Update(ctx, co)
	})
}
