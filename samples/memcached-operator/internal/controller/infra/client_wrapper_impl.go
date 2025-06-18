package infra

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8CliImpl struct {
	cli client.Client
}

func NewK8CliImpl(k8 client.Client) *K8CliImpl {
	return &K8CliImpl{k8}
}

func NewK8CliStub(e StubErrors, k8 client.Client) *K8CliStub {
	return &K8CliStub{e, k8}
}

func (k8 *K8CliImpl) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	return k8.cli.Get(ctx, t, co)
}

func (k8 *K8CliImpl) StatusUpdate(ctx context.Context, co client.Object) error {
	return k8.cli.Status().Update(ctx, co)
}

func (k8 *K8CliImpl) Create(ctx context.Context, co client.Object) error {
	return k8.cli.Create(ctx, co)
}

func (k8 *K8CliImpl) Update(ctx context.Context, co client.Object) error {
	return k8.cli.Update(ctx, co)
}

type StubErrors = map[string][]error

type K8CliStub struct {
	errArr StubErrors
	cli    client.Client
}

func (k8 *K8CliStub) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	if _, ok := k8.errArr["Get"]; ok {
		if nextErr := k8.nextErr("Get"); nextErr != nil {
			return nextErr
		}
	}

	if k8.cli != nil {
		return k8.cli.Get(ctx, t, co)
	}

	return nil
}

func (k8 *K8CliStub) nextErr(name string) error {
	if len(k8.errArr[name]) == 0 {
		fmt.Printf("no more errors configured in nulled '%s' method\n", name)
		return nil
	}

	err := k8.errArr[name][0]
	k8.errArr[name] = k8.errArr[name][1:]

	return err
}

func (k8 *K8CliStub) StatusUpdate(ctx context.Context, co client.Object) error {
	if _, ok := k8.errArr["StatusUpdate"]; ok {
		if nextErr := k8.nextErr("StatusUpdate"); nextErr != nil {
			return nextErr
		}
	}

	if k8.cli != nil {
		return k8.cli.Status().Update(ctx, co)
	}

	return nil
}

func (k8 *K8CliStub) Create(ctx context.Context, co client.Object) error {
	if _, ok := k8.errArr["Create"]; ok {
		if nextErr := k8.nextErr("Create"); nextErr != nil {
			return nextErr
		}
	}

	if k8.cli != nil {
		return k8.cli.Create(ctx, co)
	}

	return nil
}

func (k8 *K8CliStub) Update(ctx context.Context, co client.Object) error {
	if _, ok := k8.errArr["Update"]; ok {
		if nextErr := k8.nextErr("Update"); nextErr != nil {
			return nextErr
		}
	}

	if k8.cli != nil {
		return k8.cli.Update(ctx, co)
	}

	return nil
}
