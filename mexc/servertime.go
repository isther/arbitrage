package mexc

import (
	"context"
	"net/http"
)

type ServerTimeService struct {
	c *Client
}

func (s *ServerTimeService) Do(ctx context.Context, opts ...RequestOption) (servertime int64, err error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: getServerTimeApi(),
		secType:  secTypeSigned,
	}
	data, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return 0, err
	}
	j, err := newJSON(data)
	if err != nil {
		return 0, err
	}
	servertime = j.Get("servertime").MustInt64()
	return servertime, nil
}
