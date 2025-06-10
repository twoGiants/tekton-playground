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
	k8     client.Client
}

func NewClientWrapperStub(e StubErrors) *ClientWrapperStub {
	return &ClientWrapperStub{e, nil}
}

func NewClientWrapperStubWithK8(e StubErrors, k8 client.Client) *ClientWrapperStub {
	return &ClientWrapperStub{e, k8}
}

func (c *ClientWrapperStub) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	if _, ok := c.errArr["Get"]; ok {
		if nextErr := c.nextErr("Get"); nextErr != nil {
			return nextErr
		}
	}

	if c.k8 != nil {
		return c.k8.Get(ctx, t, co)
	}

	return nil
}

func (c *ClientWrapperStub) nextErr(name string) error {
	if len(c.errArr[name]) == 0 {
		fmt.Printf("no more errors configured in nulled '%s' method\n", name)
		return nil
	}

	err := c.errArr[name][0]
	c.errArr[name] = c.errArr[name][1:]

	return err
}

func (c *ClientWrapperStub) StatusUpdate(ctx context.Context, co client.Object) error {
	if _, ok := c.errArr["StatusUpdate"]; ok {
		if nextErr := c.nextErr("StatusUpdate"); nextErr != nil {
			return nextErr
		}
	}

	if c.k8 != nil {
		return c.k8.Status().Update(ctx, co)
	}

	return nil
}

func (c *ClientWrapperStub) Create(ctx context.Context, co client.Object) error {
	if _, ok := c.errArr["Create"]; ok {
		if nextErr := c.nextErr("Create"); nextErr != nil {
			return nextErr
		}
	}

	if c.k8 != nil {
		return c.k8.Create(ctx, co)
	}

	return nil
}

func (c *ClientWrapperStub) Update(ctx context.Context, co client.Object) error {
	if _, ok := c.errArr["Update"]; ok {
		if nextErr := c.nextErr("Update"); nextErr != nil {
			return nextErr
		}
	}

	if c.k8 != nil {
		return c.k8.Update(ctx, co)
	}

	return nil
}
