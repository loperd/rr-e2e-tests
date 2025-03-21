package centrifugo

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	centrifugeClient "github.com/centrifugal/centrifuge-go"
	"github.com/roadrunner-server/centrifuge/v4"
	"github.com/roadrunner-server/config/v4"
	"github.com/roadrunner-server/endure/v2"
	"github.com/roadrunner-server/logger/v4"
	"github.com/roadrunner-server/rpc/v4"
	mocklogger "github.com/roadrunner-server/rr-e2e-tests/mock"
	"github.com/roadrunner-server/server/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/exp/slog"
)

func TestCentrifugoPlugiInit(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2023.1.0",
		Path:    "configs/.rr-centrifugo-init.yaml",
		Prefix:  "rr",
	}

	cmd := exec.Command("../../env/centrifugo", "--config", "../../env/config.json", "--admin")
	err := cmd.Start()
	assert.NoError(t, err)

	go func() {
		_ = cmd.Wait()
	}()

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	_ = l
	err = cont.RegisterAll(
		cfg,
		&centrifuge.Plugin{},
		&server.Plugin{},
		&logger.Plugin{},
		&rpc.Plugin{},
		// l,
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	if err != nil {
		t.Fatal(err)
	}

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

	time.Sleep(time.Second * 5)
	client := centrifugeClient.NewProtobufClient("ws://127.0.0.1:8000/connection/websocket", centrifugeClient.Config{
		Name:               "roadrunner_tests",
		Version:            "3.0.0",
		ReadTimeout:        time.Second * 100,
		WriteTimeout:       time.Second * 100,
		HandshakeTimeout:   time.Second * 100,
		MaxServerPingDelay: time.Second * 100,
	})

	err = client.Connect()
	assert.NoError(t, err)
	time.Sleep(time.Second * 10)
	stopCh <- struct{}{}
	wg.Wait()

	require.Equal(t, 2, oLogger.FilterMessageSnippet("pipeline was started").Len())
	require.Equal(t, 2, oLogger.FilterMessageSnippet("pipeline was stopped").Len())
	require.Equal(t, 2, oLogger.FilterMessageSnippet("job was pushed successfully").Len())
	require.Equal(t, 2, oLogger.FilterMessageSnippet("job processing was started").Len())
	require.Equal(t, 2, oLogger.FilterMessageSnippet("delivery channel was closed, leaving the rabbit listener").Len())
}
