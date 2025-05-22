package infra

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientWrapperImpl struct {
	k8Client client.Client
}

func NewClientWrapper(c client.Client) *ClientWrapperImpl {
	return &ClientWrapperImpl{c}
}

func (c *ClientWrapperImpl) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	return c.k8Client.Get(ctx, t, co)
}

func (c *ClientWrapperImpl) StatusUpdate(ctx context.Context, co client.Object) error {
	return c.k8Client.Status().Update(ctx, co)
}

func (c *ClientWrapperImpl) Create(ctx context.Context, co client.Object) error {
	return c.k8Client.Create(ctx, co)
}

func (c *ClientWrapperImpl) Update(ctx context.Context, co client.Object) error {
	return c.k8Client.Update(ctx, co)
}

type StubErrors = map[string][]error

type ClientWrapperStub struct {
	errArr StubErrors
}

func NewClientWrapperStub(e StubErrors) *ClientWrapperStub {
	return &ClientWrapperStub{e}
}

func (c *ClientWrapperStub) Get(_ context.Context, _ types.NamespacedName, _ client.Object) error {
	return c.pickErrFor("Get")
}

func (c *ClientWrapperStub) pickErrFor(name string) error {
	if c.errArr == nil {
		return nil
	}

	if _, ok := c.errArr[name]; ok {
		return c.nextErr(name)
	}

	return nil
}

func (c *ClientWrapperStub) nextErr(name string) error {
	if len(c.errArr[name]) == 0 {
		panic(fmt.Sprintf("no more errors configured in nulled '%s' method\n", name))
	}

	err := c.errArr[name][0]
	c.errArr[name] = c.errArr[name][1:]

	return err
}

func (c *ClientWrapperStub) StatusUpdate(_ context.Context, _ client.Object) error {
	return c.pickErrFor("StatusUpdate")
}

func (c *ClientWrapperStub) Create(_ context.Context, _ client.Object) error {
	return c.pickErrFor("Create")
}

func (c *ClientWrapperStub) Update(ctx context.Context, _ client.Object) error {
	return c.pickErrFor("Update")
}
