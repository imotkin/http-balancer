package client

import (
	"context"
	"errors"
)

type Client struct {
	Name     string `json:"name,omitempty"`
	Key      string `json:"key,omitempty"`
	Capacity uint   `json:"capacity,omitempty"`
	Rate     uint   `json:"rate,omitempty"`
}

func (c *Client) Valid() error {
	switch {
	case c.Name == "":
		return errors.New("empty name")
	case c.Capacity == 0:
		return errors.New("null capacity")
	case c.Rate == 0:
		return errors.New("null rate")
	default:
		return nil
	}
}

type DefaultParams struct {
	Capacity uint
	Rate     uint
}

type Storage interface {
	Add(ctx context.Context, client Client) (string, error)
	Delete(ctx context.Context, key string) error
	Has(ctx context.Context, key string) (*Client, error)
	Get(ctx context.Context, key string) (*Client, error)
	List(ctx context.Context) ([]Client, error)

	Defaults() DefaultParams
}
