package grpc_test

import (
	"context"
	"crypto/tls"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/exp/slog"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/roadrunner-server/config/v4"
	"github.com/roadrunner-server/endure/v2"
	grpcPlugin "github.com/roadrunner-server/grpc/v4"
	"github.com/roadrunner-server/logger/v4"
	"github.com/roadrunner-server/resetter/v4"
	rpcPlugin "github.com/roadrunner-server/rpc/v4"
	"github.com/roadrunner-server/rr-e2e-tests/plugins/grpc/proto/service"
	"github.com/roadrunner-server/server/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestGrpcRqRsGzip(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-grpc-rq.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&grpcPlugin.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&server.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	}()

	time.Sleep(time.Second * 1)

	conn, err := grpc.Dial("127.0.0.1:9001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	require.NoError(t, err)
	require.NotNil(t, conn)

	client := service.NewEchoClient(conn)
	resp, err := client.Ping(context.Background(), &service.Message{Msg: "TOST"})
	require.NoError(t, err)
	require.Equal(t, "TOST", resp.Msg)

	stopCh <- struct{}{}

	wg.Wait()
}

func TestGrpcRqRsMultipleGzip(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-grpc-rq-multiple.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&grpcPlugin.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&server.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	}()

	time.Sleep(time.Second * 1)

	conn, err := grpc.Dial("127.0.0.1:9001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	require.NoError(t, err)
	require.NotNil(t, conn)

	client := service.NewEchoClient(conn)
	resp, err := client.Ping(context.Background(), &service.Message{Msg: "TOST"})
	require.NoError(t, err)
	require.Equal(t, "TOST", resp.Msg)

	hc := grpc_health_v1.NewHealthClient(conn)
	hr, err := hc.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, "SERVING", hr.Status.String())

	watch, err := hc.Watch(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)

	msg := &grpc_health_v1.HealthCheckResponse{}

	err = watch.RecvMsg(msg)
	require.NoError(t, err)
	require.Equal(t, "SERVING", msg.Status.String())

	err = watch.CloseSend()
	require.NoError(t, err)

	stopCh <- struct{}{}

	wg.Wait()
}

func TestGrpcRqRsTLSGzip(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-grpc-rq-tls.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&grpcPlugin.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&server.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	}()

	time.Sleep(time.Second * 1)

	cert, err := tls.LoadX509KeyPair("../../test-certs/localhost+2-client.pem", "../../test-certs/localhost+2-client-key.pem")
	require.NoError(t, err)

	tlscfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	conn, err := grpc.Dial("127.0.0.1:9002", grpc.WithTransportCredentials(credentials.NewTLS(tlscfg)), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	require.NoError(t, err)
	require.NotNil(t, conn)

	client := service.NewEchoClient(conn)
	resp, err := client.Ping(context.Background(), &service.Message{Msg: "TOST"})
	require.NoError(t, err)
	require.Equal(t, "TOST", resp.Msg)

	stopCh <- struct{}{}

	wg.Wait()
}

func TestGrpcRqRsTLSRootCAGzip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("root pool is not available on Windows")
	}
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-grpc-rq-tls-rootca.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&grpcPlugin.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&server.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	}()

	time.Sleep(time.Second * 1)

	cert, err := tls.LoadX509KeyPair("../../test-certs/localhost+2-client.pem", "../../test-certs/localhost+2-client-key.pem")
	require.NoError(t, err)

	tlscfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	conn, err := grpc.Dial("127.0.0.1:9003", grpc.WithTransportCredentials(credentials.NewTLS(tlscfg)), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	require.NoError(t, err)
	require.NotNil(t, conn)

	client := service.NewEchoClient(conn)
	resp, err := client.Ping(context.Background(), &service.Message{Msg: "TOST"})
	require.NoError(t, err)
	require.Equal(t, "TOST", resp.Msg)

	stopCh <- struct{}{}

	wg.Wait()
}

func TestGrpcRqRsTLS_WithResetGzip(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-grpc-rq-tls.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&grpcPlugin.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&server.Plugin{},
		&resetter.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	}()

	time.Sleep(time.Second * 1)

	cert, err := tls.LoadX509KeyPair("../../test-certs/localhost+2-client.pem", "../../test-certs/localhost+2-client-key.pem")
	require.NoError(t, err)

	tlscfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	conn, err := grpc.Dial("localhost:9002", grpc.WithTransportCredentials(credentials.NewTLS(tlscfg)), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	require.NoError(t, err)
	require.NotNil(t, conn)

	client := service.NewEchoClient(conn)
	resp, err := client.Ping(context.Background(), &service.Message{Msg: "TOST"})
	require.NoError(t, err)
	require.Equal(t, "TOST", resp.Msg)

	// reset
	t.Run("SendReset", sendReset("127.0.0.1:6009"))

	resp2, err2 := client.Ping(context.Background(), &service.Message{Msg: "TOST"})
	require.NoError(t, err2)
	require.Equal(t, "TOST", resp2.Msg)

	stopCh <- struct{}{}
	wg.Wait()
}
