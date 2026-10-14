package http

import (
	"context"
	"errors"
	"net"
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServerHook_PortInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	hook := serverHook(&stdhttp.Server{Addr: listener.Addr().String()}, zap.NewNop())
	err = hook.OnStart(context.Background())
	var opErr *net.OpError
	require.True(t, errors.As(err, &opErr), "expected a listener error, got %v", err)
	require.Equal(t, "listen", opErr.Op)
}
