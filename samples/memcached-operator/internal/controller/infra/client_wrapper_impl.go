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
type InfraFuncs = map[string][]client.Client

type ClientWrapperStub struct {
	errArr   StubErrors
	infraArr InfraFuncs
}

func NewClientWrapperStub(e StubErrors) *ClientWrapperStub {
	return &ClientWrapperStub{e, nil}
}

func NewClientWrapperStubWithInfra(e StubErrors, i InfraFuncs) *ClientWrapperStub {
	return &ClientWrapperStub{e, i}
}

func (c *ClientWrapperStub) Get(ctx context.Context, t types.NamespacedName, co client.Object) error {
	if client := c.pickInfraFor("Get"); client != nil {
		return client.Get(ctx, t, co)
	}

	return c.pickErrFor("Get")
}

func (c *ClientWrapperStub) pickInfraFor(name string) client.Client {
	if c.infraArr == nil {
		return nil
	}

	if _, ok := c.infraArr[name]; ok {
		return c.nextInfra(name)
	}

	return nil
}

func (c *ClientWrapperStub) nextInfra(name string) client.Client {
	if len(c.infraArr[name]) == 0 {
		panic(fmt.Sprintf("no more infra functions configured in nulled '%s' method\n", name))
	}

	client := c.infraArr[name][0]
	c.infraArr[name] = c.infraArr[name][1:]

	return client
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

func (c *ClientWrapperStub) StatusUpdate(ctx context.Context, co client.Object) error {
	if client := c.pickInfraFor("StatusUpdate"); client != nil {
		return client.Status().Update(ctx, co)
	}

	return c.pickErrFor("StatusUpdate")
}

func (c *ClientWrapperStub) Create(ctx context.Context, co client.Object) error {
	if client := c.pickInfraFor("Create"); client != nil {
		return client.Create(ctx, co)
	}

	return c.pickErrFor("Create")
}

func (c *ClientWrapperStub) Update(ctx context.Context, co client.Object) error {
	if client := c.pickInfraFor("Update"); client != nil {
		return client.Update(ctx, co)
	}

	return c.pickErrFor("Update")
}
