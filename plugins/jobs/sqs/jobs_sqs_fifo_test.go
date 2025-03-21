package sqs

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/roadrunner-server/config/v4"
	"github.com/roadrunner-server/endure/v2"
	"github.com/roadrunner-server/informer/v4"
	"github.com/roadrunner-server/jobs/v4"
	"github.com/roadrunner-server/logger/v4"
	"github.com/roadrunner-server/resetter/v4"
	rpcPlugin "github.com/roadrunner-server/rpc/v4"
	mocklogger "github.com/roadrunner-server/rr-e2e-tests/mock"
	helpers "github.com/roadrunner-server/rr-e2e-tests/plugins/jobs"
	"github.com/roadrunner-server/server/v4"
	"github.com/roadrunner-server/sqs/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/exp/slog"
)

func TestSQSInitFifo(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-sqs-init_fifo.yaml",
		Prefix:  "rr",
	}

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	err := cont.RegisterAll(
		cfg,
		&server.Plugin{},
		&rpcPlugin.Plugin{},
		l,
		&jobs.Plugin{},
		&resetter.Plugin{},
		&informer.Plugin{},
		&sqs.Plugin{},
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

	time.Sleep(time.Second * 3)
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-1", false, "127.0.0.1:6001"))
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-2", false, "127.0.0.1:6001"))
	time.Sleep(time.Second * 2)

	t.Run("DestroyPipeline", helpers.DestroyPipelines("127.0.0.1:6001", "test-1", "test-2"))

	stopCh <- struct{}{}
	wg.Wait()
	time.Sleep(time.Second)

	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("receive message").Len(), 2)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("job was pushed successfully").Len(), 2)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("job was processed successfully").Len(), 2)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("sqs listener was stopped").Len(), 2)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("destroy signal received").Len(), 1)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("pipeline was stopped").Len(), 2)
}

func TestSQSInitFifoAutoAck(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	cfg := &config.Plugin{
		Version: "2.9.2",
		Path:    "configs/.rr-sqs-init_fifo.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&server.Plugin{},
		&rpcPlugin.Plugin{},
		l,
		&jobs.Plugin{},
		&resetter.Plugin{},
		&informer.Plugin{},
		&sqs.Plugin{},
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

	time.Sleep(time.Second * 3)
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-1", true, "127.0.0.1:6001"))
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-2", true, "127.0.0.1:6001"))
	time.Sleep(time.Second * 2)

	t.Run("DestroyPipeline", helpers.DestroyPipelines("127.0.0.1:6001", "test-1", "test-2"))

	stopCh <- struct{}{}
	wg.Wait()

	require.Equal(t, 2, oLogger.FilterMessageSnippet("auto ack is turned on, message acknowledged").Len())
}

func TestSQSInitV27BadRespFifo(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Path:    "configs/.rr-sqs-init-v27-br_fifo.yaml",
		Prefix:  "rr",
		Version: "2.7.0",
	}

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	err := cont.RegisterAll(
		cfg,
		&server.Plugin{},
		&rpcPlugin.Plugin{},
		l,
		&jobs.Plugin{},
		&resetter.Plugin{},
		&informer.Plugin{},
		&sqs.Plugin{},
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

	time.Sleep(time.Second * 3)
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-1", false, "127.0.0.1:6001"))
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-2", false, "127.0.0.1:6001"))
	time.Sleep(time.Second)

	t.Run("DestroyPipeline", helpers.DestroyPipelines("127.0.0.1:6001", "test-1", "test-2"))

	stopCh <- struct{}{}
	wg.Wait()

	require.GreaterOrEqual(t, oLogger.FilterMessageSnippet("response handler error").Len(), 2)
}

func TestSQSDeclareFifo(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-sqs-declare_fifo.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&server.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&jobs.Plugin{},
		&resetter.Plugin{},
		&informer.Plugin{},
		&sqs.Plugin{},
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

	time.Sleep(time.Second * 3)

	t.Run("DeclarePipelineFifo", declareSQSPipeFifo("default-decl.fifo"))
	t.Run("ConsumePipelineFifo", helpers.ResumePipes("127.0.0.1:6001", "test-3"))
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-3", false, "127.0.0.1:6001"))
	time.Sleep(time.Second)
	t.Run("PausePipelineFifo", helpers.PausePipelines("127.0.0.1:6001", "test-3"))
	time.Sleep(time.Second)
	t.Run("DestroyPipelineFifo", helpers.DestroyPipelines("127.0.0.1:6001", "test-3"))

	time.Sleep(time.Second * 5)
	stopCh <- struct{}{}
	wg.Wait()
}

func TestSQSJobsErrorFifo(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.9.0",
		Path:    "configs/.rr-sqs-jobs-err_fifo.yaml",
		Prefix:  "rr",
	}

	err := cont.RegisterAll(
		cfg,
		&server.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&jobs.Plugin{},
		&resetter.Plugin{},
		&informer.Plugin{},
		&sqs.Plugin{},
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

	time.Sleep(time.Second * 3)

	t.Run("DeclarePipelineFifo", declareSQSPipeFifo("default-err.fifo"))
	t.Run("ConsumePipelineFifo", helpers.ResumePipes("127.0.0.1:6001", "test-3"))
	t.Run("PushPipelineFifo", helpers.PushToPipe("test-3", false, "127.0.0.1:6001"))
	time.Sleep(time.Second * 25)
	t.Run("PausePipelineFifo", helpers.PausePipelines("127.0.0.1:6001", "test-3"))
	time.Sleep(time.Second)
	t.Run("DestroyPipelineFifo", helpers.DestroyPipelines("127.0.0.1:6001", "test-3"))

	time.Sleep(time.Second * 5)
	stopCh <- struct{}{}
	wg.Wait()

	time.Sleep(time.Second * 5)
}

func TestSQSPrefetch(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2.12.1",
		Path:    "configs/.rr-sqs-init_fifo-prefetch.yaml",
		Prefix:  "rr",
	}

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	err := cont.RegisterAll(
		cfg,
		&server.Plugin{},
		&rpcPlugin.Plugin{},
		l,
		&jobs.Plugin{},
		&resetter.Plugin{},
		&informer.Plugin{},
		&sqs.Plugin{},
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

	time.Sleep(time.Second * 3)
	for i := 0; i < 10; i++ {
		go func() {
			t.Run("PushPipelineFifo", helpers.PushToPipe("test-1", false, "127.0.0.1:6232"))
			t.Run("PushPipelineFifo", helpers.PushToPipe("test-2", false, "127.0.0.1:6232"))
		}()
	}

	time.Sleep(time.Second * 40)
	t.Run("DestroyPipeline", helpers.DestroyPipelines("127.0.0.1:6232", "test-1", "test-2"))
	stopCh <- struct{}{}

	wg.Wait()

	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("prefetch limit was reached").Len(), 1)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("receive message").Len(), 2)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("job was pushed successfully").Len(), 20)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("job was processed successfully").Len(), 20)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("sqs listener was stopped").Len(), 2)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("destroy signal received").Len(), 1)
	assert.GreaterOrEqual(t, oLogger.FilterMessageSnippet("pipeline was stopped").Len(), 2)
}
