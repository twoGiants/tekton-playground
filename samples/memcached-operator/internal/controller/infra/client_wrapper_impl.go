package infra

import (
	"context"

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
	return c.k8Client.Update(ctx, co)
}

func (c *ClientWrapperImpl) Update(ctx context.Context, co client.Object) error {
	return c.k8Client.Update(ctx, co)
}

type ClientWrapperStub struct {
	errors map[string]error
}

func NewClientWrapperStub(e map[string]error) *ClientWrapperStub {
	return &ClientWrapperStub{e}
}

func (c *ClientWrapperStub) Get(_ context.Context, _ types.NamespacedName, _ client.Object) error {
	if err, ok := c.errors["Get"]; ok {
		return err
	}

	return nil
}

func (c *ClientWrapperStub) StatusUpdate(_ context.Context, _ client.Object) error {
	if err, ok := c.errors["StatusUpdate"]; ok {
		return err
	}

	return nil
}

func (c *ClientWrapperStub) Create(_ context.Context, _ client.Object) error {
	if err, ok := c.errors["Create"]; ok {
		return err
	}

	return nil
}

func (c *ClientWrapperStub) Update(ctx context.Context, _ client.Object) error {
	if err, ok := c.errors["Update"]; ok {
		return err
	}

	return nil
}
